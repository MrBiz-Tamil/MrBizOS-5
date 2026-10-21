package devi

// result.go — FlowResult and ChatAction types.
//
// Every flow method returns a FlowResult.  The HTTP handler maps this to
// ChatResponse so the frontend can render structured buttons and deep-links
// alongside the text reply.

// ChatAction is a structured button or link rendered by the frontend inside
// the chat bubble.  The frontend sends the Payload as the next user message
// when the user taps the button.
type ChatAction struct {
	Type    string `json:"type"`    // "button" | "link" | "payment" | "deep_link"
	Label   string `json:"label"`   // visible button text
	Payload string `json:"payload"` // sent as next message OR opened as URL/deep-link
}

// ChatComponent represents a rich UI element like a card, list, or tracker.
type ChatComponent struct {
	Type string                 `json:"type"` // "doctor_card", "product_card", "tracking_widget", "insurance_quote"
	Data map[string]interface{} `json:"data"`
}

// FlowResult is returned by every flow entry-point and continuation handler.
type FlowResult struct {
	// Message is the text reply shown to the user.
	Message string

	// Actions are optional structured buttons for the frontend to render.
	Actions []ChatAction

	// Components are rich UI elements (cards, widgets) for the frontend.
	Components []ChatComponent

	// Data is optional machine-readable payload for the frontend.
	Data map[string]interface{}

	// Done signals that this flow is complete and Redis state should be cleared.
	Done bool
}

// btn is a convenience constructor for a reply button.
func btn(label, payload string) ChatAction {
	return ChatAction{Type: "button", Label: label, Payload: payload}
}

// linkBtn is a convenience constructor for an external link button.
func linkBtn(label, url string) ChatAction {
	return ChatAction{Type: "link", Label: label, Payload: url}
}

// payBtn is a convenience constructor for a payment action button.
func payBtn(label, payload string) ChatAction {
	return ChatAction{Type: "payment", Label: label, Payload: payload}
}

// done returns a terminal FlowResult with no buttons.
func done(msg string) FlowResult {
	return FlowResult{Message: msg, Done: true}
}

// reply returns a non-terminal FlowResult (flow continues next turn).
func reply(msg string, actions ...ChatAction) FlowResult {
	return FlowResult{Message: msg, Actions: actions}
}
