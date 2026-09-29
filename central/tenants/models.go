package tenants

import "time"

// Tenant is the central-db account master that owns one or more organisations.
type Tenant struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	Status    string    `json:"status" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Tenant) TableName() string {
	return "tenants"
}
