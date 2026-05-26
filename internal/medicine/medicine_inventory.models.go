package medicine

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type MedPricing MedicinePricing
type MedicineInventory struct {
	ID             string     `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MedicineID     string     `json:"medicine_id" gorm:"type:uuid;not null	"`
	SupplierID     string     `json:"supplier_id" gorm:"type:uuid;not null"`
	BatchNo        string     `json:"batch_no" gorm:"type:varchar(100);not null"`
	ExpiresAt      time.Time  `json:"expires_at" gorm:"type:date;not null"`
	PurchaseQty    int        `json:"purchase_qty" gorm:"type:int;not null"`
	OrganisationID string     `json:"organisation_id" gorm:"type:uuid;not null"`
	CreatedAt      time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy      string     `json:"created_by" gorm:"type:uuid;not null"`
	Pricing        MedPricing `json:"medicine_pricing" gorm:"type:jsonb"`
}

type MedicinePricing struct {
	MRP           float64 `json:"mrp" gorm:"type:numeric(10,2);not null"`
	Discount      float64 `json:"discount" gorm:"type:numeric(10,2);not null"`
	PurchasePrice float64 `json:"purchase_price" gorm:"type:numeric(10,2);not null"`
	SellingPrice  float64 `json:"selling_price" gorm:"type:numeric(10,2);not null"`
	DiscountType  string  `json:"discount_type" gorm:"type:varchar(10);not null"`
	TotalPrice    float64 `json:"total_price" gorm:"type:numeric(10,2);not null"`
}

func (S *MedPricing) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &S)
	case string:
		return json.Unmarshal([]byte(v), &S)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}

}
func (S *MedPricing) Value() (driver.Value, error) {
	return json.Marshal(S)
}
