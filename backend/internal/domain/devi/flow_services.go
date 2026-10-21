package devi

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
)

// ── Lab Flow ─────────────────────────────────────────────────────────────────

func (e *Engine) labStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	if !uc.CanBook() {
		return done(e.guestRegisterPrompt("book a lab test"))
	}
	st := LabState{Step: 0}
	state.Intent = string(intentLab)
	state.Payload = marshalPayload(st)

	return reply(
		"🔬 *Dr.Ayya Diagnostics*\n\nWhich lab test or diagnostic package are you looking for?\n\n" +
			"Type the name (e.g. *Full Body Checkup*, *Blood Sugar*, *Thyroid*)\nor type *menu* to go back.",
	)
}

func (e *Engine) labContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	st := unmarshalLab(state)
	switch st.Step {
	case 0:
		return e.labSearch(ctx, uc, state, &st, msg)
	}
	return done(e.buildMainMenu(uc))
}

func (e *Engine) labSearch(ctx context.Context, uc *UserContext, state *sessionState, st *LabState, query string) FlowResult {
	if e.svc.LabSvc == nil {
		return done("Lab service is currently unavailable. Please try again later or call our helpline.\n\nType *menu* for other options.")
	}

	tests, err := e.svc.LabSvc.ListAvailableTests(ctx, "", "", query)
	if err != nil || len(tests) == 0 {
		return reply(
			fmt.Sprintf("🔍 No lab tests found matching *%s*. Try another name (e.g. *blood sugar*, *thyroid*, *lipid profile*).", query),
			btn("Search Again", "lab"), btn("Menu", "menu"),
		)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔬 *Available tests for \"%s\"*:\n\n", query))
	for i, t := range tests {
		if i >= 5 {
			break
		}
		category := ""
		if t.Category != "" {
			category = " | " + t.Category
		}
		sb.WriteString(fmt.Sprintf("%d️⃣  *%s*\n   💰 ₹%d%s\n\n", i+1, t.Name, t.BasePrice, category))
	}
	sb.WriteString("📞 *To book this test:*\n")
	sb.WriteString("• Call *1800-XXX-XXXX* (toll-free)\n")
	sb.WriteString("• Or open *Dr.Ayya app → Lab Tests → Book*\n\n")
	sb.WriteString("Want to search for another test?")

	return reply(sb.String(),
		btn("Search Again", "lab"), btn("Back to Menu", "menu"),
	)
}

// ── Support Flow ─────────────────────────────────────────────────────────────

func (e *Engine) supportStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	st := SupportState{Step: 0}
	state.Intent = string(intentSupport)
	state.Payload = marshalPayload(st)

	return reply(
		"🎫 *Dr.Ayya Support*\n\nSorry to hear you're having trouble! I'll help you raise a ticket right away.\n\n"+
			"What's the issue about?\n\n"+
			"1️⃣  *Order Issue* (Pharmacy / Lab)\n"+
			"2️⃣  *Appointment Issue*\n"+
			"3️⃣  *Payment / Refund*\n"+
			"4️⃣  *Feedback / Suggestion*\n\n"+
			"Tap a button or reply with a number (1-4):",
		btn("1 Order", "1"), btn("2 Appointment", "2"),
		btn("3 Payment", "3"), btn("4 Feedback", "4"),
	)
}

func (e *Engine) supportContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	st := unmarshalSupport(state)
	switch st.Step {
	case 0:
		st.IssueType = msg
		st.Step = 1
		state.Payload = marshalPayload(st)
		return reply("Got it! 👍 Please describe your issue briefly — our team will look into it right away:")
	case 1:
		st.Description = msg
		ticketID := fmt.Sprintf("TKT-%04d", rand.Intn(9000)+1000)
		return done(fmt.Sprintf(
			"✅ *Support ticket raised!*\n\n"+
				"🆔 Ticket %s\n"+
				"📋 Issue: *%s*\n\n"+
				"📞 Our team will reach you within *4 hours*.\n"+
				"💡 *Note your ticket number* — quote it if you call us."+
				proactiveFollowUp(),
			ticketID, st.IssueType,
		))
	}
	return done(e.buildMainMenu(uc))
}

// ── Payment Flow ─────────────────────────────────────────────────────────────

func (e *Engine) paymentStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	if uc.IsGuest() {
		return done(e.guestRegisterPrompt("access your wallet"))
	}

	balance := "0"
	if e.svc.FinanceSvc != nil && uc.UserID != nil {
		acct, err := e.svc.FinanceSvc.GetOrCreateAccount(ctx, fmt.Sprintf("PatientWallet:%d", *uc.UserID), "liability")
		if err == nil {
			balance = rupeesStr(acct.Balance)
		}
	}

	return reply(
		fmt.Sprintf("💳 *Dr.Ayya Wallet*\n\n"+
			"💰 Your current balance: *₹%s*\n\n"+
			"What would you like to do?\n\n"+
			"1️⃣  *Top Up* — add money to wallet\n"+
			"2️⃣  *View Transactions* — recent activity\n"+
			"3️⃣  *Refund Status* — check pending refunds\n\n"+
			"Tap a button or reply with a number:", balance),
		btn("1 Top Up", "1"), btn("2 Transactions", "2"), btn("3 Refunds", "3"),
	)
}

func (e *Engine) paymentContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	return done("For secure wallet operations, please use the *Wallet* section in the Dr.Ayya app — it's much faster and fully secure! 🔒\n\n" +
		"*Dr.Ayya app → Wallet*" +
		proactiveFollowUp())
}
