package razorpay

type Config struct {
	PaymentConfig PaymentConfig
}
type PaymentConfig struct {
	CallbackUrl    string
	RazorpayConfig RazorpayConfig
	WebhookSecret  string
}
type RazorpayConfig struct {
	ApiKey    string
	ApiSecret string
	BaseUrl   string
}

type createPaymentLinkRequest struct {
	Amount float64 `json:"amount"`

	Currency string `json:"currency"`

	AcceptPartial bool `json:"accept_partial"`

	FirstMinPartialAmount int64 `json:"first_min_partial_amount"`

	ExpireBy int64 `json:"expire_by"`

	ReferenceID string `json:"reference_id"`

	Description string `json:"description"`

	Customer customer `json:"customer"`

	Notify notify `json:"notify"`

	ReminderEnable bool `json:"reminder_enable"`

	Notes map[string]string `json:"notes"`

	CallbackURL string `json:"callback_url"`

	CallbackMethod string `json:"callback_method"`
}

type customer struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Mobile string `json:"mobile"`
}

type notify struct {
	Email bool `json:"email"`
}

type webhookEvent struct {
	Entity string `json:"entity"`
	AccountID string `json:"account_id"`
	Event string `json:"event"`
	Contains []string `json:"contains"`
	Payload webhookPayload `json:"payload"`
	CreatedAt int64 `json:"created_at"`
}

type webhookPayload struct {
	Payment webhookPayment `json:"payment"`
}

type webhookPayment struct {
	Entity webhookPaymentEntity `json:"entity"`
}

type webhookPaymentEntity struct {
	ID string `json:"id"`
	Entity string `json:"entity"`
	Amount int64 `json:"amount"`
	Currency string `json:"currency"`
	BaseAmount int64 `json:"base_amount"`
	Status string `json:"status"`
	OrderID string `json:"order_id"`
	InvoiceID *string `json:"invoice_id"`
	International bool `json:"international"`
	Method string `json:"method"`
	AmountRefunded int64 `json:"amount_refunded"`
	AmountTransferred int64 `json:"amount_transferred"`
	RefundStatus *string `json:"refund_status"`
	Captured bool `json:"captured"`
	Description *string `json:"description"`
	CardID *string `json:"card_id"`
	Bank *string `json:"bank"`
	Wallet *string `json:"wallet"`
	VPA *string `json:"vpa"`
	Email string `json:"email"`
	Contact string `json:"contact"`
	Notes []string `json:"notes"`
	Fee *int64 `json:"fee"`
	Tax *int64 `json:"tax"`
	ErrorCode *string `json:"error_code"`
	ErrorDescription *string `json:"error_description"`
	ErrorSource *string `json:"error_source"`
	ErrorStep *string `json:"error_step"`
	ErrorReason *string `json:"error_reason"`
	AcquirerData webhookAcquirerData `json:"acquirer_data"`
	CreatedAt int64 `json:"created_at"`
	UPI *webhookUPI `json:"upi"`
	Card *webhookCard `json:"card"`
	TokenID *string `json:"token_id"`
}

type webhookAcquirerData struct {
	RRN *string `json:"rrn"`
	AuthCode *string `json:"auth_code"`
}

type webhookUPI struct {
	PayerAccountType string `json:"payer_account_type"`
	VPA string `json:"vpa"`
	Flow string `json:"flow"`
}

type webhookCard struct {
	EMI bool `json:"emi"`
	Entity string `json:"entity"`
	ID string `json:"id"`
	IIN string `json:"iin"`
	International bool `json:"international"`
	Issuer *string `json:"issuer"`
	Last4 string `json:"last4"`
	Name string `json:"name"`
	Network string `json:"network"`
	SubType string `json:"sub_type"`
	Type string `json:"type"`
}
