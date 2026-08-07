package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type MvmtStatus string
type SourceTypes string

var (
	PurchaseEntry        SourceTypes = "purchase_entry"
	PatientMedicineOrder SourceTypes = "patient_medicine_order"
)

var (
	Sale     MvmtStatus = "sale"
	Purchase MvmtStatus = "purchase"
	Return   MvmtStatus = "return"
	Dispense MvmtStatus = "dispense"
)

type MedicineStockMovements struct {
	ID                    string      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MedicineID            string      `json:"medicine_id" gorm:"type:uuid;not null"`
	MedicineInventoryID   string      `json:"medicine_inventory_id" gorm:"type:uuid;not null"`
	OrganisationID        string      `json:"organisation_id" gorm:"type:uuid;not null"`
	MovementType          MvmtStatus  `json:"movement_type" gorm:"type:varchar(100);not null"`
	QtyChanged            int         `json:"qty_changed" gorm:"type:int;not null"`
	CreatedBy             string      `json:"created_by" gorm:"type:uuid;not null"`
	SourceType            SourceTypes `json:"source_type" gorm:"type:varchar(100);not null"`
	UnitPriceAtTimeOfMvmt float64     `json:"unit_price_at_time_of_mvmt" gorm:"type:numeric(10,2);not null"`
	BalanceAfterMvmt      int         `json:"balance_after_mvmt" gorm:"type:int;not null"`
	CreatedAt             time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
}

type Pricing struct {
	MRP           float64 `json:"mrp" gorm:"type:numeric(10,2);default:0"`
	UnitPrice     float64 `json:"unit_price" gorm:"type:numeric(10,2);default:0"`
	Discount      float64 `json:"discount" gorm:"type:numeric(10,2);default:0"`
	PurchasePrice float64 `json:"purchase_price" gorm:"type:numeric(10,2);default:0"`
	SellingPrice  float64 `json:"selling_price" gorm:"type:numeric(10,2);default:0"`
	DiscountType  string  `json:"discount_type" gorm:"type:varchar(100);default:'amount'"`
	TotalPrice    float64 `json:"total_price" gorm:"type:numeric(10,2);default:0"`
}

func (p *Pricing) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	}
	return fmt.Errorf("unsupported type: %T", value)
}
func (p *Pricing) Value() (driver.Value, error) {
	return json.Marshal(p)
}
