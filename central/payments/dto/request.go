package dto

// CreateOrderRequest is the payload central sends to the internal order API.
// Amount is in currency subunits (paise for INR).
type CreateOrderRequest struct {
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"` // ISO 4217, e.g. INR
	Receipt  string            `json:"receipt"`
	Notes    map[string]string `json:"notes"`
}

// CreateOrderResponse mirrors the Razorpay Orders API create response.
// https://razorpay.com/docs/api/orders/create/
type CreateOrderResponse struct {
	ID         string `json:"id"`
	Entity     string `json:"entity"`
	Amount     int64  `json:"amount"`
	AmountPaid int64  `json:"amount_paid"`
	AmountDue  int64  `json:"amount_due"`
	Currency   string `json:"currency"`
	Receipt    string `json:"receipt,omitempty"`
	OfferID    *string `json:"offer_id"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts"`
	Notes      any    `json:"notes,omitempty"`
	CreatedAt  int64  `json:"created_at"`
}

// APIErrorBody is the error JSON returned by the internal payment API.
type APIErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
