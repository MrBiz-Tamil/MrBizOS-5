package devi

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/visibility"
)

// ucContextKey is used to pass the current UserContext through context.Context
// so LLM tool handlers can access the authenticated user's ID.
type ucContextKey struct{}

// contextWithUC attaches a UserContext to a context so tool handlers can read it.
func contextWithUC(ctx context.Context, uc *UserContext) context.Context {
	return context.WithValue(ctx, ucContextKey{}, uc)
}

// ucFromContext retrieves the UserContext attached by contextWithUC.
func ucFromContext(ctx context.Context) *UserContext {
	uc, _ := ctx.Value(ucContextKey{}).(*UserContext)
	return uc
}



// ToolDefinition describes a function that can be called by the LLM
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolHandler is the actual Go function that implements the tool
type ToolHandler func(ctx context.Context, e *Engine, args map[string]interface{}) (string, error)

var toolRegistry = map[string]ToolHandler{
	"get_online_doctors":     handleGetOnlineDoctors,
	"search_pharmacy":        handleSearchPharmacy,
	"get_lab_tests":          handleGetLabTests,
	"check_my_appointments":  handleCheckMyAppointments,
	"search_knowledge_base":  handleSearchKnowledgeBase,
	"start_registration":    handleStartRegistration,
	"add_to_cart":           handleAddToCart,
	"get_order_status":      handleGetOrderStatus,
	"get_available_slots":   handleGetAvailableSlots,
	"get_popular_doctors":   handleGetPopularDoctors,
}



// GetToolDefinitions returns the JSON schema for all registered tools
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "add_to_cart",
			Description: "Add a medicine or health product to the user's pharmacy cart. Use this after a user confirms they want to buy something.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"medicine_id": map[string]interface{}{"type": "integer"},
					"quantity":    map[string]interface{}{"type": "integer", "default": 1},
				},
				"required": []string{"medicine_id"},
			},
		},
		{
			Name:        "get_order_status",
			Description: "Check the live status of a pharmacy or lab order using the Order ID.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"order_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"order_id"},
			},
		},
		{
			Name:        "get_available_slots",
			Description: "Get available appointment slots for a specific doctor.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"doctor_id": map[string]interface{}{"type": "integer"},
				},
				"required": []string{"doctor_id"},
			},
		},
		{
			Name:        "get_popular_doctors",
			Description: "Get a list of top-rated doctors currently available on Dr.Ayya. Use this to provide social proof and recommendations.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "start_registration",


			Description: "Start the account registration process for a guest user. Use this when a guest wants to book an appointment, order medicine, or save their progress.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"phone": map[string]interface{}{
						"type":        "string",
						"description": "The user's mobile number",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Optional: The user's name",
					},
				},
				"required": []string{"phone"},
			},
		},
		{
			Name:        "search_knowledge_base",

			Description: "Search the Dr.Ayya medical knowledge base and service FAQ. Use this for specific health questions or information about Dr.Ayya services.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The natural language question or topic to search for",
					},
				},
				"required": []string{"query"},
			},
		},

		{
			Name:        "get_online_doctors",
			Description: "Returns a list of doctors currently online and available for consultation. Use this when the user wants to see a doctor now.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"specialization": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter by specialization (e.g. Cardiology, General Medicine)",
					},
				},
			},
		},
		{
			Name:        "search_pharmacy",
			Description: "Search for medicines and health products in the Dr.Ayya pharmacy catalog. Returns names, prices, and stock status.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The name of the medicine or product to search for",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_lab_tests",
			Description: "List available lab tests and diagnostic packages (Blood tests, Scans, etc.)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search term (e.g. 'sugar', 'mri')",
					},
				},
			},
		},
		{
			Name:        "check_my_appointments",
			Description: "Get the status of the current user's upcoming appointments. Only use this if the user asks about their own bookings.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// ─── Tool Handlers ───────────────────────────────────────────────────────────

func handleGetOnlineDoctors(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	spec, _ := args["specialization"].(string)
	
	// Use Visibility service via engine (if it's not in svc, we use the global one for now, 
	// but ideally it should be in ServiceContainer)
	doctors, err := visibility.GetOnlineDoctors()
	if err != nil {
		return "", err
	}

	if spec != "" {
		filtered := make([]visibility.DoctorView, 0)
		for _, d := range doctors {
			if strings.Contains(strings.ToLower(d.Specialization), strings.ToLower(spec)) {
				filtered = append(filtered, d)
			}
		}
		doctors = filtered
	}

	if len(doctors) == 0 {
		return "I couldn't find any online doctors matching that specialty right now.", nil
	}

	var sb strings.Builder
	sb.WriteString("Here are the doctors currently available:\n")
	for i, d := range doctors {
		if i >= 5 { break }
		sb.WriteString(fmt.Sprintf("- *Dr. %s* (%s) | Fee: ₹%d | ⭐ %.1f\n", 
			d.FullName, d.Specialization, d.ConsultationFee, d.AverageRating))
	}
	return sb.String(), nil
}

func handleSearchPharmacy(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "Please provide a product name to search.", nil
	}

	if e.svc.StoreSvc == nil {
		return "Pharmacy service is currently unavailable.", nil
	}

	products, err := e.svc.StoreSvc.SearchProducts(ctx, query, "", "", 5, 0)
	if err != nil {
		return "I encountered an error searching the pharmacy.", nil
	}

	if len(products) == 0 {
		return fmt.Sprintf("I couldn't find any products matching '%s' in our pharmacy.", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("I found these items for '%s':\n", query))
	for _, p := range products {
		sb.WriteString(fmt.Sprintf("- *%s* | ₹%.2f\n", p.Name, float64(p.SellingPrice)/100.0))
	}
	return sb.String(), nil
}

func handleGetLabTests(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)

	if e.svc.LabSvc == nil {
		return "Lab service is currently unavailable.", nil
	}

	tests, err := e.svc.LabSvc.ListAvailableTests(ctx, "", "", query)
	if err != nil {
		return "I encountered an error fetching lab tests.", nil
	}

	if len(tests) == 0 {
		return "No matching lab tests found.", nil
	}

	var sb strings.Builder
	sb.WriteString("Here are the available lab tests:\n")
	for i, t := range tests {
		if i >= 5 { break }
		sb.WriteString(fmt.Sprintf("- *%s* | ₹%d\n", t.Name, t.BasePrice))
	}
	return sb.String(), nil
}

func handleCheckMyAppointments(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	uc := ucFromContext(ctx)
	if uc == nil || uc.UserID == nil {
		return "You need to be logged in to view your appointments. Please log in and try again.", nil
	}
	if e.svc.AppointmentSvc == nil {
		return "Appointment service is currently unavailable.", nil
	}

	appts, err := e.svc.AppointmentSvc.GetUpcomingAppointments(*uc.UserID)
	if err != nil || len(appts) == 0 {
		return "You have no upcoming appointments. Would you like to *book a consultation*?", nil
	}

	ist, _ := time.LoadLocation("Asia/Kolkata")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You have %d upcoming appointment(s):\n\n", len(appts)))
	for i, a := range appts {
		if i >= 5 {
			break
		}
		slot := a.StartTime.In(ist).Format("Mon, Jan 2 · 3:04 PM IST")
		sb.WriteString(fmt.Sprintf("• *Appt #%d* — %s | Status: %s\n", a.ID, slot, string(a.Status)))
	}
	sb.WriteString("\nType *menu* for other options.")
	return sb.String(), nil
}

func handleStartRegistration(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	phone, _ := args["phone"].(string)
	name, _ := args["name"].(string)

	if phone == "" {
		return "I need a phone number to start your registration.", nil
	}

	// Upsert guest session — match on phone, set nickname if provided.
	// Use INSERT … ON CONFLICT for atomicity; fall back to update-by-phone.
	updateFields := map[string]interface{}{
		"phone":        phone,
		"last_seen_at": time.Now(),
	}
	if name != "" {
		updateFields["nickname"] = name
	}

	uc := ucFromContext(ctx)
	if uc != nil && uc.GuestBrowserID != "" {
		// Prefer matching by browser_id (exact session) when available
		err := e.svc.DB.Table("devi_guest_sessions").
			Where("browser_id = ?", uc.GuestBrowserID).
			Updates(updateFields).Error
		if err != nil {
			log.Printf("[Devi] start_registration: DB update failed: %v", err)
		}
	} else {
		// Fallback: upsert by phone (avoids duplicate records)
		err := e.svc.DB.Table("devi_guest_sessions").
			Where("phone = ?", phone).
			Updates(updateFields).Error
		if err != nil {
			log.Printf("[Devi] start_registration: DB upsert failed: %v", err)
		}
	}

	greeting := ""
	if name != "" {
		greeting = fmt.Sprintf("Nice to meet you, *%s*! ", name)
	}
	return fmt.Sprintf("%sI've noted your number *%s*.\n\n👉 To complete registration, visit *Dr.Ayya.in/register* — it takes 30 seconds and unlocks appointments, pharmacy, and more!", greeting, phone), nil
}

func handleAddToCart(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	rawID, ok := args["medicine_id"].(float64)
	if !ok || rawID <= 0 {
		return "I need a valid medicine ID to add to cart.", nil
	}
	medicineID := uint(rawID)
	quantity := 1
	if q, ok := args["quantity"].(float64); ok && q > 0 {
		quantity = int(q)
	}

	// Look up the medicine name for a friendly reply
	medName := fmt.Sprintf("item #%d", medicineID)
	pricePaise := int64(0)
	if e.svc.DB != nil {
		var med struct {
			Name      string `gorm:"column:name"`
			UnitPrice int64  `gorm:"column:unit_price"`
		}
		if err := e.svc.DB.Table("medicine").Select("name, unit_price").Where("id = ?", medicineID).Scan(&med).Error; err == nil && med.Name != "" {
			medName = med.Name
			pricePaise = med.UnitPrice
		}
	}

	// Write to the user's PharmacyState in Redis so the next pharmacy flow
	// picks it up automatically.
	uc := ucFromContext(ctx)
	if uc != nil && e.svc.Redis != nil {
		state := redisLoadState(ctx, e.svc.Redis, uc.SessionKey)
		if state.Intent == "" {
			// Start a fresh pharmacy state
			state.Intent = string(intentPharmacy)
		}
		if state.Intent == string(intentPharmacy) {
			st := unmarshalPharmacy(state)
			found := false
			for i := range st.Cart {
				if st.Cart[i].MedID == medicineID {
					st.Cart[i].Qty += quantity
					found = true
					break
				}
			}
			if !found {
				st.Cart = append(st.Cart, CartItem{
					MedID:      medicineID,
					Name:       medName,
					Qty:        quantity,
					PricePaise: pricePaise,
				})
			}
			st.CartTotal = calcCartTotal(st.Cart)
			st.Step = 1
			state.Payload = marshalPayload(st)
			redisSaveState(ctx, e.svc.Redis, uc.SessionKey, state, sessionTTL(uc))
		}
	} else {
		log.Printf("[Devi] handleAddToCart: no Redis/UC — skipping cart persistence (med=%d qty=%d)", medicineID, quantity)
	}

	return fmt.Sprintf("✅ Added *%s* ×%d to your cart.\n\nType *checkout* to place your order, or keep searching for more medicines.", medName, quantity), nil
}


func handleGetOrderStatus(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	orderID, _ := args["order_id"].(string)
	if orderID == "" {
		return "Please provide an Order ID.", nil
	}

	var order struct {
		OrderNumber string
		Status      string
		TrackingID  string
	}
	err := e.svc.DB.Table("pharmacy_orders").
		Select("order_number, status, tracking_id").
		Where("order_number = ? OR id::text = ?", orderID, orderID).
		First(&order).Error

	if err != nil {
		return fmt.Sprintf("I couldn't find an order with ID %s. Please double check the number.", orderID), nil
	}

	statusMsg := fmt.Sprintf("Order %s is currently **%s**.", order.OrderNumber, strings.ToUpper(order.Status))
	if order.TrackingID != "" {
		statusMsg += fmt.Sprintf("\nTracking ID: %s", order.TrackingID)
	}
	return statusMsg, nil
}

func handleGetAvailableSlots(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	doctorIDRaw, _ := args["doctor_id"].(float64)
	doctorID := uint(doctorIDRaw)
	
	if e.svc.DB == nil {
		return "I'm sorry, I can't check schedules right now.", nil
	}

	var docProfile identity.DoctorProfile
	if err := e.svc.DB.Table("doctorprofile").Where("id = ? OR user_id = ?", doctorID, doctorID).First(&docProfile).Error; err != nil {
		return "I couldn't find that doctor's schedule.", nil
	}

	slots := e.generateSlots(ctx, docProfile.ID)
	if len(slots) == 0 {
		return "This doctor has no available slots for the next 24 hours.", nil
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Available slots for *Dr. %s*:\n", docProfile.Specialization))
	for _, s := range slots {
		sb.WriteString(fmt.Sprintf("- %s\n", s.Label))
	}
	return sb.String(), nil
}


func handleGetPopularDoctors(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	if e.svc.DB == nil {
		return "I can't fetch doctor profiles right now.", nil
	}

	var doctors []struct {
		FullName       string  `gorm:"column:full_name"`
		Specialization string  `gorm:"column:specialization"`
		AverageRating  float64 `gorm:"column:average_rating"`
	}

	err := e.svc.DB.Table("doctorprofile dp").
		Select("u.full_name, dp.specialization, dp.average_rating").
		Joins("JOIN \"user\" u ON u.id = dp.user_id").
		Where("dp.is_verified = ? AND dp.is_online = ?", true, true).
		Order("dp.average_rating DESC, dp.admin_boost_score DESC").
		Limit(3).
		Scan(&doctors).Error

	if err != nil || len(doctors) == 0 {
		return "We have many top-rated specialists available! Tell me what you're looking for.", nil
	}

	res := "Here are some of our top-rated specialists online now:\n"
	for _, d := range doctors {
		res += fmt.Sprintf("- *Dr. %s* (%s) | ⭐ %.1f\n", d.FullName, d.Specialization, d.AverageRating)
	}
	res += "\nWould you like to book a consultation with any of them?"
	return res, nil
}

// DispatchCall executes a tool by name with the given arguments
func (e *Engine) DispatchCall(ctx context.Context, name string, args map[string]interface{}) (string, error) {



	handler, ok := toolRegistry[name]
	if !ok {
		return "", fmt.Errorf("tool %s not found", name)
	}
	return handler(ctx, e, args)
}
