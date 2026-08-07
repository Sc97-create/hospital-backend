package razorpay

import "encoding/json"

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

type paymentLinkResponse struct {
	ID          string `json:"id"`
	ShortURL    string `json:"short_url"`
	ReferenceID string `json:"reference_id"`
	Status      string `json:"status"`
}

type createPaymentLinkRequest struct {
	Amount                int64             `json:"amount"`
	Currency              string            `json:"currency"`
	AcceptPartial         bool              `json:"accept_partial"`
	FirstMinPartialAmount int64             `json:"first_min_partial_amount,omitempty"`
	ExpireBy              int64             `json:"expire_by,omitempty"`
	ReferenceID           string            `json:"reference_id"`
	Description           string            `json:"description"`
	Customer              customer          `json:"customer"`
	Notify                notify            `json:"notify"`
	ReminderEnable        bool              `json:"reminder_enable"`
	Notes                 map[string]string `json:"notes,omitempty"`
	CallbackURL           string            `json:"callback_url,omitempty"`
	CallbackMethod        string            `json:"callback_method,omitempty"`
	UPILink               bool              `json:"upi_link,omitempty"`
}

type customer struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Contact string `json:"contact"`
}

type notify struct {
	Email bool `json:"email"`
}

type WebhookEvent struct {
	Entity    string         `json:"entity"`
	AccountID string         `json:"account_id"`
	Event     string         `json:"event"`
	Contains  []string       `json:"contains"`
	Payload   webhookPayload `json:"payload"`
	CreatedAt int64          `json:"created_at"`
}

type webhookPayload struct {
	Order       webhookOrder       `json:"order"`
	Payment     webhookPayment     `json:"payment"`
	PaymentLink webhookPaymentLink `json:"payment_link"`
}

type webhookOrder struct {
	Entity webhookOrderEntity `json:"entity"`
}

type webhookOrderEntity struct {
	Amount         int64           `json:"amount"`
	AmountDue      int64           `json:"amount_due"`
	AmountPaid     int64           `json:"amount_paid"`
	Attempts       int             `json:"attempts"`
	Authorized     bool            `json:"authorized"`
	CreatedAt      int64           `json:"created_at"`
	Currency       string          `json:"currency"`
	ID             string          `json:"id"`
	MerchantID     string          `json:"merchant_id"`
	Method         *string         `json:"method"`
	Notes          json.RawMessage `json:"notes"`
	PartialPayment bool            `json:"partial_payment"`
	PaymentCapture bool            `json:"payment_capture"`
	ProductID      string          `json:"product_id"`
	ProductType    string          `json:"product_type"`
	Receipt        string          `json:"receipt"`
	Status         string          `json:"status"`
	UpdatedAt      int64           `json:"updated_at"`
}

type webhookPaymentLink struct {
	Entity webhookPaymentLinkEntity `json:"entity"`
}

type webhookPaymentLinkEntity struct {
	AcceptPartial         bool              `json:"accept_partial"`
	Amount                int64             `json:"amount"`
	AmountPaid            int64             `json:"amount_paid"`
	CallbackMethod        string            `json:"callback_method"`
	CallbackURL           string            `json:"callback_url"`
	CancelledAt           int64             `json:"cancelled_at"`
	CreatedAt             int64             `json:"created_at"`
	Currency              string            `json:"currency"`
	Customer              webhookCustomer   `json:"customer"`
	Description           string            `json:"description"`
	ExpireBy              int64             `json:"expire_by"`
	ExpiredAt             int64             `json:"expired_at"`
	FirstMinPartialAmount int64             `json:"first_min_partial_amount"`
	ID                    string            `json:"id"`
	Notes                 json.RawMessage   `json:"notes"`
	Notify                webhookLinkNotify `json:"notify"`
	OrderID               string            `json:"order_id"`
	ReferenceID           string            `json:"reference_id"`
	ReminderEnable        bool              `json:"reminder_enable"`
	Reminders             webhookReminders  `json:"reminders"`
	ShortURL              string            `json:"short_url"`
	Status                string            `json:"status"`
	UpdatedAt             int64             `json:"updated_at"`
	UPILink               bool              `json:"upi_link"`
	UserID                string            `json:"user_id"`
	WhatsappLink          bool              `json:"whatsapp_link"`
}

type webhookCustomer struct {
	Contact string `json:"contact"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

type webhookLinkNotify struct {
	Email    bool `json:"email"`
	SMS      bool `json:"sms"`
	Whatsapp bool `json:"whatsapp"`
}

type webhookReminders struct {
	Status string `json:"status"`
}

type webhookPayment struct {
	Entity webhookPaymentEntity `json:"entity"`
}

type webhookPaymentEntity struct {
	AcquirerData      webhookAcquirerData `json:"acquirer_data"`
	Amount            int64               `json:"amount"`
	AmountCaptured    *int64              `json:"amount_captured"`
	AmountRefunded    int64               `json:"amount_refunded"`
	AmountTransferred int64               `json:"amount_transferred"`
	Bank              *string             `json:"bank"`
	BaseAmount        int64               `json:"base_amount"`
	Captured          bool                `json:"captured"`
	Card              *webhookCard        `json:"card"`
	CardID            *string             `json:"card_id"`
	Contact           string              `json:"contact"`
	CreatedAt         int64               `json:"created_at"`
	Currency          string              `json:"currency"`
	Description       *string             `json:"description"`
	Email             *string             `json:"email"`
	Entity            string              `json:"entity"`
	ErrorCode         *string             `json:"error_code"`
	ErrorDescription  *string             `json:"error_description"`
	ErrorReason       *string             `json:"error_reason"`
	ErrorSource       *string             `json:"error_source"`
	ErrorStep         *string             `json:"error_step"`
	Fee               *int64              `json:"fee"`
	FeeBearer         string              `json:"fee_bearer"`
	ID                string              `json:"id"`
	International     bool                `json:"international"`
	InvoiceID         *string             `json:"invoice_id"`
	Method            string              `json:"method"`
	Notes             json.RawMessage     `json:"notes"`
	OrderID           string              `json:"order_id"`
	Provider          *string             `json:"provider"`
	RefundStatus      *string             `json:"refund_status"`
	Reward            *string             `json:"reward"`
	Status            string              `json:"status"`
	Tax               *int64              `json:"tax"`
	TokenID           *string             `json:"token_id"`
	UPI               *webhookUPI         `json:"upi"`
	VPA               *string             `json:"vpa"`
	Wallet            *string             `json:"wallet"`
}

type webhookAcquirerData struct {
	RRN      *string `json:"rrn"`
	AuthCode *string `json:"auth_code"`
}

type webhookUPI struct {
	PayerAccountType string `json:"payer_account_type"`
	VPA              string `json:"vpa"`
	Flow             string `json:"flow"`
}

type webhookCard struct {
	EMI           bool    `json:"emi"`
	Entity        string  `json:"entity"`
	ID            string  `json:"id"`
	IIN           string  `json:"iin"`
	International bool    `json:"international"`
	Issuer        *string `json:"issuer"`
	Last4         string  `json:"last4"`
	Name          string  `json:"name"`
	Network       string  `json:"network"`
	SubType       string  `json:"sub_type"`
	Type          string  `json:"type"`
}
