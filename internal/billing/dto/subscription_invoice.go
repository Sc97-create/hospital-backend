package dto

// SubscriptionInvoiceRequest is the Basic Auth payload for a subscription invoice.
// billing_id is the subscription id. Renewal false creates subscription_new; true creates subscription_renewal.
type SubscriptionInvoiceRequest struct {
	SubscriptionID string  `json:"subscription_id"`
	CustomerID     string  `json:"customer_id"`
	OrganisationID string  `json:"organisation_id"`
	Amount         float64 `json:"amount"`
	Renewal        bool    `json:"renewal"`
}

// SubscriptionInvoiceResponse is the created invoice.
type SubscriptionInvoiceResponse struct {
	InvoiceID   string `json:"invoice_id"`
	PaymentType string `json:"payment_type"`
	Status      string `json:"status"`
}
