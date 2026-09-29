package subscriptions

import "time"

// Subscription is the tenant-scoped entitlement for a plan.
type Subscription struct {
	ID                string     `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID          string     `json:"tenant_id" gorm:"type:uuid;not null;index"`
	PlanID            string     `json:"plan_id" gorm:"type:uuid;not null;index"`
	Status            string     `json:"status" gorm:"type:text;not null"`
	BillingCycle      string     `json:"billing_cycle" gorm:"type:varchar(50)"`
	Price             float64    `json:"price" gorm:"type:numeric(12,2);not null;default:0"`
	OrderID           string     `json:"order_id,omitempty" gorm:"column:order_id;type:varchar(100);index"`
	ProviderPaymentID string     `json:"provider_payment_id,omitempty" gorm:"column:provider_payment_id;type:varchar(100)"`
	PaymentSignature  string     `json:"payment_signature,omitempty" gorm:"column:payment_signature;type:varchar(255)"`
	StartAt           *time.Time `json:"start_at,omitempty" gorm:"type:timestamp"`
	EndAt             *time.Time `json:"end_at,omitempty" gorm:"type:timestamp"`
	CreatedAt         time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}
