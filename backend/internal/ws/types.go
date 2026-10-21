package ws

type MessageType string

const (
	TypeChat           MessageType = "text"
	TypeJoin           MessageType = "join_room"
	TypeLeave          MessageType = "leave_room"
	TypeTypingStart    MessageType = "typing_start"
	TypeTypingStop     MessageType = "typing_stop"
	TypeReadReceipt    MessageType = "read_receipt"   // client sends: I read up to message X
	TypeMessagesRead   MessageType = "messages_read"  // server broadcasts: messages marked read
	TypeCatchUp        MessageType = "catch_up"        // server sends missed messages on reconnect
	TypeRoomList       MessageType = "room_list"       // server sends updated room list
	TypeNewRoom        MessageType = "new_room"         // server notifies about a newly created room
	TypeSOSAlert       MessageType = "sos_alert"       // broadcast to all online doctors
	TypeSOSAccepted    MessageType = "sos_accepted"    // doctor accepted the SOS
	TypeDoctorEnRoute   MessageType = "doctor_en_route" // doctor is travelling to patient
	TypeDoctorArrived   MessageType = "doctor_arrived"  // doctor reached patient location
	TypeCourierLocation MessageType = "courier_location_update" // courier GPS telemetry
	TypeWalletUpdate    MessageType = "wallet_update"   // server notifies client of ledger balance change
)

// WSMsg represents the standard message format from client
type WSMsg struct {
	Type      string `json:"type"` // join_room, text, typing, read_receipt, etc.
	RoomID    string `json:"room_id"`
	Message   string `json:"message,omitempty"`
	FileURL   string `json:"file_url,omitempty"`
	SenderID  string `json:"sender_id,omitempty"`
	MessageID uint   `json:"message_id,omitempty"` // used by read_receipt
	Lat       float64 `json:"lat,omitempty"`
	Lng       float64 `json:"lng,omitempty"`
}

// WebSocketMessage represents the broadcast payload
type WebSocketMessage struct {
	Type       interface{}            `json:"type"`
	RoomID     string                 `json:"room_id"`
	SenderID   uint                   `json:"sender_id"`
	SenderName string                 `json:"sender_name"`
	SenderRole string                 `json:"sender_role"`
	Content    string                 `json:"content,omitempty"`
	Timestamp  string                 `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
