package models

// TransferCompletedEvent represents the event payload when a transfer is completed
// This matches the AsyncAPI schema defined in services/common/analytics-service-kafka-spec/asyncapi.yaml
type TransferCompletedEvent struct {
	EventID        string `json:"eventId"`
	EventType      string `json:"eventType"`
	EventTimestamp string `json:"eventTimestamp"`
	OperationID    string `json:"operationId"`
	SenderID       string `json:"senderId"`
	RecipientID    string `json:"recipientId"`
	Amount         Amount `json:"amount"`
	IdempotencyKey string `json:"idempotencyKey"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
	Message        string `json:"message,omitempty"` // Optional field
}
