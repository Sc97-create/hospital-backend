package dto

import (
	"time"
)

type CreatePaymentCommand struct {
	InvoiceID      string            `json:"invoice_id"`
	PatientID      string            `json:"patient_id"`
	InitiatedBy    string            `json:"initiated_by"`
	Source         string            `json:"source"`
	Channel        string            `json:"channel"`
	Amount         float64           `json:"amount"`
	ReferenceID    string            `json:"reference_id"` // invoice code sent to Razorpay as reference_id
	Currency       string            `json:"currency"`
	Customer       CustomerInfo      `json:"customer"`
	Description    string            `json:"description"`
	ExpiresAt      time.Time         `json:"expires_at"`
	Metadata       map[string]string `json:"metadata"`
	SendSMS        bool              `json:"send_sms"`
	SendEmail      bool              `json:"send_email"`
	PrescriptionID string            `json:"prescription_id"`
	IdempotencyKey string            `json:"idempotency_key"` // required from frontend
	//CallbackURL string
}
type CustomerInfo struct {
	Name   string
	Email  string
	Mobile string
}
type CreatePaymentResponse struct {
	RequestPayload map[string]interface{} `json:"request_payload,omitempty"`
	PaymentLinkID  string                 `json:"payment_link_id"` // Razorpay plink_xxx
	PaymentURL     string                 `json:"payment_url"`     // Razorpay short_url
	ReferenceID    string                 `json:"reference_id"`
}

type GetPaymentRequest struct {
	PaymentID string `json:"payment_id"`
}

type GetPaymentResponse struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}

type UpdatePaymentRequest struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}

type UpdatePaymentResponse struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}

// payments/dto/request.go
type ParsedWebhookEvent struct {
	EventType         string  // e.g. "payment_link.paid"
	ProviderEventID   string  // razorpay account_id or similar
	ProviderLinkID    string  // plink_xxx
	ProviderOrderID   string  // order_xxx
	ProviderPaymentID string  // pay_xxx
	ReferenceID       string  // your invoice code
	AmountPaid        float64 // in rupees (divide by 100)
	AmountTransferred float64
	PaymentStatus     string // "captured", "failed", etc.
	PaymentLinkStatus string // "paid", "cancelled", etc.
	PayerVPA          string
	PayerAccountType  string
	PaymentError      *string
	PaymentErrorCode  *string
	RawPayload        []byte // full raw JSON for storage
	PaidAt            time.Time
	AcceptPartial     bool
}
