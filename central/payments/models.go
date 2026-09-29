package payments

import "time"

// PaymentOrder is a provider order created for a central payment flow (e.g. subscription).
type PaymentOrder struct {
	ID              string            `json:"id" gorm:"type:uuid;primaryKey"`
	ProviderOrderID string            `json:"provider_order_id" gorm:"type:varchar(100);not null;uniqueIndex"`
	Amount          int64             `json:"amount" gorm:"type:bigint;not null"` // smallest currency unit (paise)
	Currency        string            `json:"currency" gorm:"type:varchar(3);not null"`
	Receipt         string            `json:"receipt,omitempty" gorm:"type:varchar(40)"`
	Status          string            `json:"status" gorm:"type:varchar(50);not null"`
	Notes           map[string]string `json:"notes,omitempty" gorm:"serializer:json"`
	CreatedAt       time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

func (PaymentOrder) TableName() string {
	return "payment_orders"
}
