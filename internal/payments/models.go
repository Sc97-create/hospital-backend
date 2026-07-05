package payments

import (
	"time"

	"gorm.io/datatypes"
)

type Payments struct {
	ID             string    `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceID      string    `json:"invoice_id" gorm:"type:uuid;not null"`
	PatientID      string    `json:"patient_id" gorm:"type:uuid;not null"`
	Amount         float64   `json:"amount" gorm:"type:numeric(10,2);not null"`
	Currency       string    `json:"currency" gorm:"type:varchar(5);not null;default:INR"`
	Source         string    `json:"source" gorm:"type:varchar(50);not null;default:link"`
	Channel        string    `json:"channel" gorm:"type:varchar(50);not null;default:upi"`
	InitiatedBy    string    `json:"initiated_by" gorm:"type:varchar(50);not null"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"type:varchar(255);not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
type PaymentAttempts struct {
	ID                  string            `json:"id" gorm:"type:uuid;not null;primaryKey"`
	ProviderPaymentID   string            `json:"provider_payment_id" gorm:"type:varchar(50);"` //payment.id
	PaymentID           string            `json:"payment_id" gorm:"type:uuid;not null"`
	Provider            string            `json:"provider" gorm:"type:varchar(50);not null"`
	AttemptNo           int               `json:"attempt_no" gorm:"type:int;not null"`
	ProviderLinkID      string            `json:"provider_link_id" gorm:"type:varchar;index"` // Razorpay plink_xxx
	ProviderOrderID     string            `json:"provider_order_id" gorm:"type:varchar"`      // webhook order_id
	PaidAt              time.Time         `json:"paid_at" gorm:"type:timestamptz"`
	PaymentLink         string            `json:"payment_link" gorm:"type:text"`
	ProviderRequest     datatypes.JSONMap `json:"provider_request" gorm:"type:jsonb"`
	PayerAccountType    string            `json:"payer_account_type" gorm:"type:varchar(50);"` //payer.account_type
	PaymentVPA          string            `json:"payment_vpa" gorm:"type:varchar(255);"`
	AcceptPartial       bool              `json:"accept_partial" gorm:"type:boolean;default:false"`
	AmountPaid          float64           `json:"amount_paid" gorm:"type:numeric(10,2);"`          //payer.amount
	AmountTransferred   float64           `json:"amount_transferred" gorm:"type:numeric(10,2);"`   //payment.amount_transferred
	PaymentError        string            `json:"payment_error" gorm:"type:text"`                  //payment.error
	PaymentErrorCode    string            `json:"payment_error_code" gorm:"type:varchar(50);"`     //payment.error_code
	ProviderReferenceID string            `json:"provider_reference_id" gorm:"type:varchar(255);"` //payment.reference_id
	//WebhookResponse   datatypes.JSONMap `json:"webhook_response" gorm:"type:jsonb"`           //on webhook arrival
	PaymentLinkStatus string     `json:"payment_link_status" gorm:"type:varchar(50);"` //webhook change status
	PaymentStatus     string     `json:"payment_status" gorm:"type:varchar(50);"`      //webhook change status
	ExpiresAt         *time.Time `json:"expires_at" gorm:"type:timestamptz"`
	CreatedAt         time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

type Refunds struct {
	ID               string            `json:"id" gorm:"type:uuid;not null;primaryKey"`
	PaymentAttemptID string            `json:"payment_attempt_id" gorm:"type:uuid;not null"`
	ProviderRefundID string            `json:"provider_refund_id" gorm:"type:varchar;not null"`
	Amount           float64           `json:"amount" gorm:"type:numeric(10,2);not null"`
	Reason           string            `json:"reason" gorm:"type:text"`
	Status           string            `json:"status" gorm:"type:varchar(50);not null"`
	ProviderData     datatypes.JSONMap `json:"provider_data" gorm:"type:jsonb"`
	GatewayResponse  datatypes.JSONMap `json:"gateway_response" gorm:"type:jsonb"`
	CreatedAt        time.Time         `json:"created_at" gorm:"autoCreateTime"`
}

type WebhookEvents struct {
	ID               string            `json:"id" gorm:"type:uuid;default:gen_random_uuid();not null;primaryKey"`
	PaymentAttemptID string            `json:"payment_attempt_id" gorm:"type:uuid;"`
	EventType        string            `json:"event_type" gorm:"type:varchar(50);not null"`
	ProviderResponse datatypes.JSONMap `json:"provider_response" gorm:"type:jsonb"`
	CreatedAt        time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}
