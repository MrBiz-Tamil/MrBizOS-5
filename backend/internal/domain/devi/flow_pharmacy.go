package devi

// flow_pharmacy.go — Medicine search → cart → checkout flow.
//
// Step 0: User types search query → show top-5 results
// Step 1: User selects item → show cart + ask qty / add more / checkout
// Step 2: Cart review → ask shipping address
// Step 3: Order confirmation → storeSvc.CreateOrder()

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/utils/privacy"
)

// ── Entry point ───────────────────────────────────────────────────────────────

func (e *Engine) pharmacyStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	if !uc.CanAccessPharmacy() {
		return done(e.guestRegisterPrompt("order medicines"))
	}
	st := PharmacyState{Step: 0}
	state.Intent = string(intentPharmacy)
	state.Payload = marshalPayload(st)

	if strings.TrimSpace(msg) != "" && !isExitCommand(strings.ToLower(msg)) {
		// Extract the specific product name from the message.
		// Generic phrases like "I want to buy medicines" return "" → fall through to show menu.
		// Specific phrases like "buy paracetamol" return "paracetamol" → search immediately.
		if product := extractProductQuery(msg); product != "" {
			return e.pharmacySearch(ctx, uc, state, &st, product)
		}
		// Fall through — show the pharmacy greeting/prompt.
	}

	return reply(
		"💊 *Dr.Ayya Pharmacy*\n\nWhat medicine or health product are you looking for?\n\n"+
			"Type the name (e.g. *Paracetamol*, *Vitamin D*, *BP monitor*)\nor type *menu* to go back.",
	)
}

func (e *Engine) pharmacyContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	st := unmarshalPharmacy(state)
	switch st.Step {
	case 0:
		return e.pharmacySearch(ctx, uc, state, &st, msg)
	case 1:
		return e.pharmacySelect(ctx, uc, state, &st, msg)
	case 2:
		return e.pharmacyCartReview(ctx, uc, state, &st, msg) // fix: was calling pharmacyShowCart (loop)
	case 3:
		return e.pharmacyConfirm(ctx, uc, state, &st, msg)
	}
	return done(e.buildMainMenu(uc))
}

// ── Step 0: search ────────────────────────────────────────────────────────────

func (e *Engine) pharmacySearch(ctx context.Context, uc *UserContext, state *sessionState, st *PharmacyState, query string) FlowResult {
	st.SearchQuery = query

	products, err := searchMedicines(e, query, 5)
	if err != nil || len(products) == 0 {
		return reply(
			fmt.Sprintf("🔍 No results found for *%s*.\n\nTry a different name, or type *menu* to go back.", query),
			btn("Try Again", query), btn("Menu", "menu"),
		)
	}

	// Convert to MedCards
	cards := make([]MedCard, 0, len(products))
	for _, p := range products {
		if p.Medicine.ID == 0 {
			continue
		}
		cards = append(cards, MedCard{
			ID:           p.Medicine.ID,
			Name:         p.Medicine.Name,
			Manufacturer: p.Medicine.Manufacturer,
			PricePaise:   p.Medicine.Prices["INR"],
			InStock:      p.StockQuantity > 0,
		})
	}

	st.Results = cards
	st.Step = 1
	state.Payload = marshalPayload(st)

	msg := fmt.Sprintf("🔍 *Search results for \"%s\"*:\n\n", query)
	actions := []ChatAction{}
	for i, c := range cards {
		stock := "✅ In stock"
		if !c.InStock {
			stock = "❌ Out of stock"
		}
		msg += fmt.Sprintf("%d️⃣  *%s*\n   💊 %s  |  💰 ₹%s  |  %s\n\n",
			i+1, c.Name, c.Manufacturer, rupeesStr(c.PricePaise), stock)
		actions = append(actions, btn(fmt.Sprintf("%d %s", i+1, c.Name), fmt.Sprintf("%d", i+1)))
	}
	msg += "Reply with a number to add to cart, or type another medicine name to search again.\nType *checkout* to place order | *menu* to cancel."

	return reply(msg, actions...)
}

// ── Step 1: item selection / cart management ──────────────────────────────────

func (e *Engine) pharmacySelect(ctx context.Context, uc *UserContext, state *sessionState, st *PharmacyState, msg string) FlowResult {
	lower := strings.ToLower(strings.TrimSpace(msg))

	// Checkout shortcut
	if lower == "checkout" || lower == "order" || lower == "buy" {
		if len(st.Cart) == 0 {
			return reply("Your cart is empty. Please add items first.\n\nType a medicine name to search.", btn("Menu", "menu"))
		}
		return e.pharmacyShowCart(ctx, uc, state, st)
	}

	// Numeric selection
	selected := -1
	for i := range st.Results {
		if msg == fmt.Sprintf("%d", i+1) {
			selected = i
			break
		}
	}

	// Handle qty change request — "qty 2", "quantity 3", etc.
	if strings.HasPrefix(lower, "qty ") || strings.HasPrefix(lower, "quantity ") {
		qtyStr := strings.TrimPrefix(strings.TrimPrefix(lower, "qty "), "quantity ")
		var qty int
		if _, err := fmt.Sscanf(qtyStr, "%d", &qty); err == nil && qty >= 1 && qty <= 20 && len(st.Cart) > 0 {
			last := &st.Cart[len(st.Cart)-1]
			last.Qty = qty
			st.CartTotal = calcCartTotal(st.Cart)
			state.Payload = marshalPayload(st)
			return reply(
				fmt.Sprintf("✅ Quantity updated to ×%d for *%s*.\n\n🛒 Cart: %d item(s) — ₹%s total\n\nSearch for another medicine or type *checkout* to order.",
					qty, last.Name, len(st.Cart), rupeesStr(st.CartTotal)),
				btn("Checkout", "checkout"), btn("Add More", st.SearchQuery), btn("Menu", "menu"),
			)
		}
		return reply("Please type *qty* followed by a number between 1–20, e.g. *qty 2*")
	}

	if selected == -1 {
		// Re-search with the new query
		return e.pharmacySearch(ctx, uc, state, st, msg)
	}

	item := st.Results[selected]
	if !item.InStock {
		return reply(
			fmt.Sprintf("❌ *%s* is currently out of stock.\n\nChoose another item or type *menu* to go back.", item.Name),
			btn("Search Again", st.SearchQuery),
		)
	}

	// Add to cart (qty 1 for now)
	found := false
	for i, ci := range st.Cart {
		if ci.MedID == item.ID {
			st.Cart[i].Qty++
			found = true
			break
		}
	}
	if !found {
		st.Cart = append(st.Cart, CartItem{
			MedID:      item.ID,
			Name:       item.Name,
			Qty:        1,
			PricePaise: item.PricePaise,
		})
	}
	st.CartTotal = calcCartTotal(st.Cart)
	state.Payload = marshalPayload(st)

	return reply(
		fmt.Sprintf("✅ *%s* added to cart (×1).\n💡 Type *qty 2* to change quantity.\n\n🛒 Cart: %d item(s) — ₹%s total\n\nSearch for another medicine or type *checkout* to order.",
			item.Name, len(st.Cart), rupeesStr(st.CartTotal)),
		btn("Checkout", "checkout"), btn("Add More", st.SearchQuery), btn("Menu", "menu"),
	)
}

// ── Step 2: cart review → ask address ────────────────────────────────────────

func (e *Engine) pharmacyShowCart(ctx context.Context, uc *UserContext, state *sessionState, st *PharmacyState) FlowResult {
	msg := "🛒 *Your Cart*\n\n"
	for _, ci := range st.Cart {
		msg += fmt.Sprintf("• %s × %d — ₹%s\n", ci.Name, ci.Qty, rupeesStr(ci.PricePaise*int64(ci.Qty)))
	}
	msg += fmt.Sprintf("\n💰 *Total: ₹%s*\n\n", rupeesStr(st.CartTotal))
	msg += "Please type your *delivery address* to continue, or type *cancel* to go back."

	st.Step = 2
	state.Payload = marshalPayload(st)

	return reply(msg, btn("Cancel", "cancel"))
}

func (e *Engine) pharmacyCartReview(ctx context.Context, uc *UserContext, state *sessionState, st *PharmacyState, msg string) FlowResult {
	lower := strings.TrimSpace(msg)
	if lower == "" || len(lower) < 5 {
		return reply("Please enter a valid delivery address (minimum 5 characters).")
	}

	// Store address in cart metadata via a temporary field
	// We use a JSON hack: store address in the cart's search query field temporarily
	st.SearchQuery = lower // reuse for address (will be cleared after order)
	st.Step = 3
	state.Payload = marshalPayload(st)

	cartSummary := ""
	for _, ci := range st.Cart {
		cartSummary += fmt.Sprintf("• %s × %d — ₹%s\n", ci.Name, ci.Qty, rupeesStr(ci.PricePaise*int64(ci.Qty)))
	}

	return reply(
		fmt.Sprintf("📦 *Order Summary*\n\n%s\n💰 *Total: ₹%s*\n📍 Deliver to: %s\n\n"+
			"Reply *YES* to confirm or *cancel* to go back.",
			cartSummary, rupeesStr(st.CartTotal), lower),
		btn("Confirm Order", "yes"), btn("Cancel", "cancel"),
	)
}

// ── Step 3: confirm → CreateOrder ────────────────────────────────────────────

func (e *Engine) pharmacyConfirm(ctx context.Context, uc *UserContext, state *sessionState, st *PharmacyState, msg string) FlowResult {
	if !isYes(strings.ToLower(strings.TrimSpace(msg))) {
		return done("Order cancelled. Type *pharmacy* to start again or *menu* for other options.")
	}

	if uc.UserID == nil {
		return done(e.guestRegisterPrompt("place a pharmacy order"))
	}

	// Rx gate: check if any cart item requires a prescription
	if e.svc.DB != nil {
		medIDs := make([]uint, 0, len(st.Cart))
		for _, ci := range st.Cart {
			medIDs = append(medIDs, ci.MedID)
		}
		if len(medIDs) > 0 {
			var rxMeds []pharmacy.Medicine
			e.svc.DB.WithContext(ctx).
				Where("id IN ? AND is_prescription_required = ?", medIDs, true).
				Find(&rxMeds)
			if len(rxMeds) > 0 {
				names := make([]string, 0, len(rxMeds))
				for _, m := range rxMeds {
					names = append(names, "*"+m.Name+"*")
				}
				return done("⚠️ *Prescription Required*\n\n" +
					strings.Join(names, ", ") + " require a valid prescription.\n\n" +
					"Please upload your prescription in the *Dr.Ayya app → Pharmacy → Upload Prescription*, " +
					"then place the order from there.\n\n" +
					"Type *menu* for other options.")
			}
		}
	}

	address := st.SearchQuery // was repurposed for address in step 2

	// Build order items JSON
	type orderItem struct {
		MedicineID uint  `json:"medicine_id"`
		Quantity   int   `json:"quantity"`
		UnitPrice  int64 `json:"unit_price"`
	}
	items := make([]orderItem, len(st.Cart))
	for i, ci := range st.Cart {
		items[i] = orderItem{MedicineID: ci.MedID, Quantity: ci.Qty, UnitPrice: ci.PricePaise}
	}
	itemsJSON, _ := json.Marshal(items)

	// Construct order payload for logging or future use (but not used in current CreateOrder call)
	_ = &pharmacy.PharmacyOrder{
		OrderNumber:     fmt.Sprintf("DEVI-%d", time.Now().Unix()),
		UserID:          *uc.UserID,
		Items:           itemsJSON,
		Amount:     st.CartTotal,
		Status:          "pending",
		PaymentStatus:   "pending",
		PaymentMethod:   "cod",
		DeliveryMode:    pharmacy.ModeStandard,
		ShippingAddress: privacy.EncryptedString(address),
	}

	if e.svc.StoreSvc != nil {
		storeItems := make([]store.OrderItemRequest, len(items))
		for i, it := range items {
			storeItems[i] = store.OrderItemRequest{ProductID: it.MedicineID, Quantity: it.Quantity}
		}
		_, err := e.svc.StoreSvc.CreateOrder(ctx, *uc.UserID, storeItems, address, 1, "", "")
		if err != nil {
			return done("⚠️ We couldn't place your order right now. Please try from the Dr.Ayya app.\n\nType *menu* for other options.")
		}
		orderNum := fmt.Sprintf("DEVI-%d", time.Now().Unix()%100000)
		return done(fmt.Sprintf(
			"🎉 *Order Placed Successfully!*\n\n"+
				"🆔 Order #%s\n"+
				"🛒 %d item(s) | ₹%s total\n"+
				"📍 Delivering to: %s\n\n"+
				"💳 Pay on delivery (COD)\n"+
				"🚚 Expected delivery: 2–4 hours\n\n"+
				"Track your order in the *Dr.Ayya app* → Orders."+
				proactiveFollowUp(),
			orderNum, len(st.Cart), rupeesStr(st.CartTotal), address,
		))
	}

	// StoreSvc not wired yet — graceful degradation
	orderNum := fmt.Sprintf("DEVI-%d", time.Now().Unix()%100000)
	return done(fmt.Sprintf(
		"🎉 *Order Request Received!*\n\n"+
			"🆔 Order #%s\n"+
			"🛒 %d item(s) | ₹%s total\n"+
			"📍 Delivering to: %s\n\n"+
			"👉 Complete your order in the *Dr.Ayya app → Pharmacy* section."+
			proactiveFollowUp(),
		orderNum, len(st.Cart), rupeesStr(st.CartTotal), address,
	))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// searchMedicines calls the store's SearchProducts function.
// Falls back to a raw DB query if StoreSvc is nil.
func searchMedicines(e *Engine, query string, limit int) ([]pharmacy.PharmacyInventory, error) {
	if e.svc.DB == nil {
		return nil, nil
	}
	var results []pharmacy.PharmacyInventory
	like := "%" + query + "%"
	err := e.svc.DB.
		Joins("Medicine").
		Where("pharmacyinventory.is_active = ? AND pharmacyinventory.stock_quantity > ?", true, 0).
		Where("Medicine.name ILIKE ? OR Medicine.brand ILIKE ? OR Medicine.category ILIKE ?", like, like, like).
		Preload("Medicine").
		Limit(limit).
		Find(&results).Error
	return results, err
}

func calcCartTotal(cart []CartItem) int64 {
	var total int64
	for _, ci := range cart {
		total += ci.PricePaise * int64(ci.Qty)
	}
	return total
}
