package medicine

import (
	"time"
)

type Medicine struct {
	ID          string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Form        string    `json:"form" gorm:"type:varchar(50);not null"`
	Strength    string    `json:"strength" gorm:"type:varchar(50);not null"`
	BatchNumber string    `json:"batch_number" gorm:"type:varchar(100);not null"`
	ExpiryDate  time.Time `json:"expiry_date" gorm:"type:date;not null"`
	Quantity    int64     `json:"quantity" gorm:"not null;check:quantity >= 0"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	CreatedBy   string    `json:"created_by" gorm:"type:uuid;not null"`
}
type InventoryLogs struct {
	ID         string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MedicineID string    `json:"medicine_id" gorm:"type:varchar(255);not null"`
	ChangeQty  int       `json:"change_qty" gorm:"type:varchar(100);not null"`
	Reason     string    `json:"reason" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	CreatedBy  string    `json:"created_by" gorm:"type:uuid;not null"`
}
