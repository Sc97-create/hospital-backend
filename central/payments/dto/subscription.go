package dto

// SubscriptionInvoiceRequest is sent to the hospital internal invoice API.
type SubscriptionInvoiceRequest struct {
	SubscriptionID string  `json:"subscription_id"`
	CustomerID     string  `json:"customer_id"`
	OrganisationID string  `json:"organisation_id"`
	Amount         float64 `json:"amount"`
	Renewal        bool    `json:"renewal"`
}

// SubscriptionInvoiceResponse is returned by the internal invoice API.
type SubscriptionInvoiceResponse struct {
	InvoiceID   string `json:"invoice_id"`
	PaymentType string `json:"payment_type"`
	Status      string `json:"status"`
}

// SubscriptionPaymentRequest is sent to the hospital internal payment API.
type SubscriptionPaymentRequest struct {
	InvoiceID  string  `json:"invoice_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	OrderID    string  `json:"order_id"`
	Currency   string  `json:"currency"`
}

// SubscriptionPaymentResponse is returned by the internal payment API.
type SubscriptionPaymentResponse struct {
	PaymentID string `json:"payment_id"`
	OrderID   string `json:"order_id"`
}

// ConfirmSubscriptionPaymentRequest stores the Checkout payment id and signature.
type ConfirmSubscriptionPaymentRequest struct {
	OrderID           string `json:"order_id"`
	ProviderPaymentID string `json:"provider_payment_id"`
	PaymentSignature  string `json:"payment_signature"`
}

// ConfirmSubscriptionPaymentResponse identifies the payment and its subscription.
type ConfirmSubscriptionPaymentResponse struct {
	PaymentID      string `json:"payment_id"`
	OrderID        string `json:"order_id"`
	SubscriptionID string `json:"subscription_id"`
}
