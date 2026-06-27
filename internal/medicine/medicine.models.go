package medicine

import (
	"time"
)

type CodePrefix string

var (
	MedicineCodePrefix CodePrefix = "MED"
	BatchCodePrefix    CodePrefix = "BAT"
)

type Medicine struct {
	ID             string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code           string    `json:"code" gorm:"type:varchar(50);uniqueIndex"`
	Name           string    `json:"name" gorm:"type:varchar(255);not null"`
	Form           string    `json:"form" gorm:"type:varchar(50);not null"`
	Strength       string    `json:"strength" gorm:"type:varchar(50);not null"`
	IsActive       bool      `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy      string    `json:"created_by" gorm:"type:uuid;not null"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	HSNCode        string    `json:"hsn_code" gorm:"type:text"`
	ReorderLevel   int       `json:"reorder_level" gorm:"type:int;not null;default:50"`     // Trigger threshold in individual units (tablets)
	MaxStockTarget int       `json:"max_stock_target" gorm:"type:int;not null;default:200"` // Target stock in individual units (tablets)
}
