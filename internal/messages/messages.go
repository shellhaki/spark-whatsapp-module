package messages

type DeliveryRequest struct {
	Text string `json:"text"`
}

type DirectDeliveryRequest struct {
	PhoneNumber string `json:"phone_number"`
	Text        string `json:"text"`
}

type DeliveryResult struct {
	JID       string `json:"jid"`
	MessageID string `json:"message_id"`
}
