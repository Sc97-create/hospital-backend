package payments

import (
	"time"

	"gorm.io/datatypes"
)

type Payments struct {
	ID             string    `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceID      string    `json:"invoice_id" gorm:"type:uuid;not null"`
	PatientID      string    `json:"patient_id" gorm:"type:uuid;not null"`
	Amount         float64   `json:"amount" gorm:"type:number"`
	Currency       string    `json:"currency" gorm:"default:INR"`
	Source         string    `json:"source" gorm:"default:link"`
	Channel        string    `json:"channel" gorm:"default:upi"`
	InitiatedBy    string    `json:"initiated_by" gorm:"type:uuid; not null"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"type:uuid"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoCreateTime"`
}
type PaymentAttempts struct {
	ID                string            `json:"id" gorm:"type:uuid;not null;primaryKey"`
	ProviderPaymentID string            `json:"provider_payment_id"`
	PaymentID         string            `json:"payment_id" gorm:"type:uuid;not null"`
	Provider          string            `json:"provider"`
	AttemptNo         int               `json:"attempt_no"`
	ProviderOrderID   string            `json:"provider_order_id" gorm:"type:varchar"` //webhook
	PaidAt            time.Time         `json:"paid_at" gorm:"type:timestamptz"`
	PaymentLink       string            `json:"payment_link" gorm:"type:text"`
	ProviderRequest   datatypes.JSONMap `json:"provider_request" gorm:"type:jsonb"`
	GatewayResponse   datatypes.JSONMap `json:"gateway_response" gorm:"type:jsonb"` //on webhook arrival
	Status            string            `json:"status"`                             //webhook change status
	ExpiresAt         *time.Time        `json:"expires_at" gorm:"type:timestamptz"`
	CreatedAt         time.Time         `json:"created_at" gorm:"autoCreateTime"`
}

type Refunds struct {
	ID               string            `json:"id" gorm:"type:uuid;not null;primaryKey"`
	PaymentAttemptID string            `json:"payment_attempt_id" gorm:"type:uuid;not null"`
	ProviderRefundID string            `json:"provider_refund_id" gorm:"type:varchar"`
	Amount           float64           `json:"amount" gorm:"type:number"`
	Reason           string            `json:"reason" gorm:"type:text"`
	Status           string            `json:"status"`
	ProviderData     datatypes.JSONMap `json:"provider_data" gorm:"type:jsonb"`
	GatewayResponse  datatypes.JSONMap `json:"gateway_response" gorm:"type:jsonb"`
	CreatedAt        time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}
