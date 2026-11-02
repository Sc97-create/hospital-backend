package dto

// SubscriptionPaymentRequest stores a card payment opened from a Razorpay order.
// source is link and channel is card. customer_id is required. patient_id is omitted.
type SubscriptionPaymentRequest struct {
	InvoiceID  string  `json:"invoice_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	OrderID    string  `json:"order_id"`
	Currency   string  `json:"currency"`
}

// SubscriptionPaymentResponse is the stored payment.
type SubscriptionPaymentResponse struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
}

// ConfirmSubscriptionPaymentRequest stores the Checkout payment id and signature on the payment.
type ConfirmSubscriptionPaymentRequest struct {
	OrderID           string `json:"order_id"`
	ProviderPaymentID string `json:"provider_payment_id"`
	PaymentSignature  string `json:"payment_signature"`
}

// ConfirmSubscriptionPaymentResponse identifies the payment and the subscription on its invoice.
type ConfirmSubscriptionPaymentResponse struct {
	PaymentID      string `json:"payment_id"`
	OrderID        string `json:"order_id"`
	SubscriptionID string `json:"subscription_id"`
}
