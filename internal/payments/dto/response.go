package dto

type PaymentResponse struct {
	RequestPayload map[string]interface{} `json:"request_payload"`
	PaymentID      string                 `json:"payment_id"`
	PaymentURL     string                 `json:"payment_url"`
}
