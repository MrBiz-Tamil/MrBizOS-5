package devi

// flow_greeting.go — Main menu, unknown-intent handler, register prompt,
// premium upsell, and account overview.  These are stateless (Done: true).

import "fmt"

// proactiveFollowUp returns a warm closing line appended after successful task completions.
func proactiveFollowUp() string {
	return "\n\n✨ Is there anything else I can help you with?\nType *menu* to see all options."
}

func (e *Engine) greetingStart(uc *UserContext) FlowResult {
	return done(e.buildMainMenu(uc))
}

func (e *Engine) buildMainMenu(uc *UserContext) string {
	var header string
	switch {
	case uc.IsFamilyMember() && uc.FamilyMemberName != "":
		header = fmt.Sprintf("👶 Checking on *%s* today? I'm here to help! 💙\n\nHere's what I can do:", uc.FamilyMemberName)
	case uc.IsGuest():
		header = "👋 Hi there! I'm *Devi*, your personal health companion at Dr.Ayya. 😊\n\n" +
			"You can ask me anything — symptoms, medicines, or general health questions!\n" +
			"💡 *Create a free account* to book doctors, order medicines & much more (just 30 seconds).\n\n" +
			"Here's what's available for you right now:"
	default:
		header = fmt.Sprintf("👋 Hey %s! Great to see you. 😊\nI'm *Devi* — your personal health companion at Dr.Ayya.\n\nHow can I help you today?", uc.DisplayName)
	}

	menu := "\n\n🩺  *Symptom Check* — describe how you feel\n"
	menu += "💊  *Medicine Info* — dosage, side effects, interactions\n"
	if uc.CanBook() {
		menu += "📅  *Book Appointment* — consult a doctor now\n"
		menu += "🛒  *Pharmacy* — order medicines & health products\n"
		menu += "🔬  *Lab Tests* — book diagnostics\n"
		menu += "💳  *Wallet* — balance, top-up, pay for consultation\n"
		menu += "📋  *My Account* — appointments, orders, prescriptions\n"
	} else {
		menu += "📅  *Book Appointment* — 🔒 free account needed\n"
		menu += "🛒  *Pharmacy* — 🔒 free account needed\n"
		menu += "🔬  *Lab Tests* — 🔒 free account needed\n"
	}
	menu += "🎫  *Support* — raise a ticket\n"
	menu += "🛡️  *Insurance* — explore health plans\n"
	if uc.CanBook() {
		menu += "⭐  *Go Premium* — unlimited consultations\n"
	}
	menu += "\nJust type what you need — or ask me anything in plain words!\nType *0* or *menu* anytime to come back here."

	return header + menu
}

func (e *Engine) registerStart() FlowResult {
	return done("📝 *Create your free Dr.Ayya account*\n\n" +
		"It's quick and completely free! Here's how:\n\n" +
		"1️⃣  Visit *Dr.Ayya.in* or download the app\n" +
		"2️⃣  Tap *Sign Up* → enter your phone number\n" +
		"3️⃣  Verify OTP → set a password\n\n" +
		"✅ *Done in under 60 seconds!*\n\n" +
		"Once you're in, I can book doctor appointments, order medicines, run lab tests, and keep your health records safe. 💙\n\n" +
		"Type *menu* anytime to explore what's available.")
}

func (e *Engine) guestRegisterPrompt(action string) string {
	return fmt.Sprintf("💙 To *%s*, you'll need a free Dr.Ayya account.\n\n"+
		"Good news — it's completely free and takes just 30 seconds!\n\n"+
		"✅ Book doctors in real-time\n"+
		"✅ Order medicines with home delivery\n"+
		"✅ Save your medical history securely\n\n"+
		"👉 Visit *Dr.Ayya.in/register* to sign up\n\n"+
		"Already have an account? Log in and come back — I'll be right here! 😊", action)
}

func (e *Engine) premiumStart(uc *UserContext) FlowResult {
	if uc.IsGuest() {
		return done(e.guestRegisterPrompt("upgrade to Prime"))
	}
	return done("⭐ *Dr.Ayya Prime*\n\n" +
		"Upgrade for just *₹199/month*:\n\n" +
		"✅ Unlimited video consultations\n" +
		"✅ Priority doctor matching (< 2 min)\n" +
		"✅ Free medicines up to ₹500/month\n" +
		"✅ Free lab tests (2 per month)\n" +
		"✅ Dedicated health manager\n" +
		"✅ Family coverage (up to 4 members)\n\n" +
		"💡 *Average saving: ₹3,200/month* vs paying per consultation\n\n" +
		"👉 Tap *Go Prime* in the app or visit the *Subscriptions* section.\n\n" +
		"Type *menu* for other options.")
}

func (e *Engine) unknownStart(msg string) FlowResult {
	hint := suggestExample(msg)
	return done(fmt.Sprintf(
		"Hmm, I'm not quite sure what you meant by *\"%s\"*. 🤔\n\n"+
			"Here are some things you can try:\n\n"+
			"🩺  *I have a fever* → symptom check\n"+
			"💊  *What is ibuprofen* → drug info\n"+
			"📅  *Book a doctor* → appointments\n"+
			"🛒  *Buy paracetamol* → pharmacy\n"+
			"🎫  *I need help* → support ticket\n\n"+
			"💡 %s\n\n"+
			"Or type *menu* to see everything I can do.", msg, hint))
}

func (e *Engine) myAccountStart(ctx interface{}, uc *UserContext) FlowResult {
	if uc.IsGuest() {
		return done(e.guestRegisterPrompt("view your account"))
	}
	return done("📋 *My Account*\n\n" +
		"Here's a quick overview of your Dr.Ayya profile:\n\n" +
		"1️⃣  Upcoming appointments\n" +
		"2️⃣  Past appointments & prescriptions\n" +
		"3️⃣  Pharmacy orders\n" +
		"4️⃣  Lab test reports\n\n" +
		"👉 Open the *Dr.Ayya app* for the full experience — manage appointments, view reports, and track orders in one place.\n\n" +
		"Need something specific right now? I can help you *book*, *order medicines*, or *check your wallet*!\n" +
		"Type *menu* for all options.")
}
