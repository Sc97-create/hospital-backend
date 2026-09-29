package customers

import "time"

// Customer is a central-db account used for tenant/subscription onboarding.
type Customer struct {
	ID              string     `json:"id" gorm:"type:uuid;primaryKey"`
	FullName        string     `json:"full_name" gorm:"type:varchar(255);not null"`
	WorkEmail       string     `json:"work_email" gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash    string     `json:"-" gorm:"column:password_hash;type:text;not null"`
	Status          string     `json:"status" gorm:"type:text;not null"`
	TenantID        *string    `json:"tenant_id,omitempty" gorm:"column:tenant_id;type:uuid;index"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" gorm:"type:timestamp"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Customer) TableName() string {
	return "customers"
}

// SignupCode stores a one-time email verification code for a customer.
// TenantID is optional at signup (tenant is created later in the onboarding flow).
type SignupCode struct {
	ID         string     `json:"id" gorm:"type:uuid;primaryKey"`
	UserID     string     `json:"user_id" gorm:"column:user_id;type:uuid;not null;index"` // customer_id
	Code       string     `json:"code" gorm:"type:varchar(6);not null"`
	TenantID   *string    `json:"tenant_id,omitempty" gorm:"column:tenant_id;type:uuid;index"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"type:timestamp;not null"`
	VerifiedAt *time.Time `json:"verified_at,omitempty" gorm:"type:timestamp"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SignupCode) TableName() string {
	return "customer_signup_codes"
}
