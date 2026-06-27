package medicine

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type MedPricing MedicinePricing
type MedicineInventory struct {
	ID             string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MedicineID     string    `json:"medicine_id" gorm:"type:uuid;not null;index"`
	SupplierID     string    `json:"supplier_id" gorm:"type:uuid;not null"`
	BatchNo        string    `json:"batch_no" gorm:"type:varchar(100);not null"`
	ExpiresAt      time.Time `json:"expires_at" gorm:"type:date;not null;index"` // INDEX added for FEFO (Fast Expiry sorting)
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null;index"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy      string    `json:"created_by" gorm:"type:uuid;not null"`

	// --- ADDED FOR INVOICE LINKING ---
	PurchaseEntryID string `json:"purchase_entry_id" gorm:"type:uuid;not null;index"` // Connects batch directly to the B2B Wholesale Invoice

	// --- ADDED FOR UNIT-LEVEL MATHEMATICAL INTEGRITY ---
	PurchaseQtyBoxes  int `json:"purchase_qty_boxes" gorm:"type:int;not null"`  // How many full boxes/strips were bought
	UnitsPerBox       int `json:"units_per_box" gorm:"type:int;not null"`       // Saved snapshot of pack size at purchase time
	CurrentStockUnits int `json:"current_stock_units" gorm:"type:int;not null"` // Remaining loose units (e.g., individual tablets); purchase_qty_boxes * unit_per_box

	// --- ADDED FOR PHYSICAL HUMAN VERIFICATION ("Shelf GPS") ---
	ShelfLocation string `json:"shelf_location" gorm:"type:varchar(100);default:'Unassigned'"` // e.g., "Rack 4-A"

	// --- JSONB PRICING STRUCTURE ---
	Pricing MedicinePricing `json:"medicine_pricing" gorm:"type:jsonb;not null"`
}
type MixedMedInventory struct {
	ID                string          `json:"id"`
	MedicineID        string          `json:"medicine_id"`
	BatchNo           string          `json:"batch_no"`
	ExpiresAt         time.Time       `json:"expiry_at"`
	MedicineName      string          `json:"medicine_name"`
	MedForm           string          `json:"med_form"`
	MedicineStrength  string          `json:"medicine_strength"`
	ShelfLocation     string          `json:"shelf_location"`
	PurchaseQtyBoxes  int             `json:"purchase_qty_boxes"`
	UnitsPerBox       int             `json:"units_per_box"`
	CurrentStockUnits int             `json:"current_stock_units"`
	Pricing           MedicinePricing `json:"medicine_pricing"`
}

type MedicinePricing struct {
	MRP           float64 `json:"mrp" gorm:"type:numeric(10,2);not null"`
	UnitPrice     float64 `json:"unit_price" gorm:"type:numeric(10,2)"` // mrp / units_per_box  => units_per_box: in one box how many tablets are there; 30/15=2.00
	Discount      float64 `json:"discount" gorm:"type:numeric(10,2);not null"`
	PurchasePrice float64 `json:"purchase_price" gorm:"type:numeric(10,2);not null"`
	SellingPrice  float64 `json:"selling_price" gorm:"type:numeric(10,2);not null"`
	DiscountType  string  `json:"discount_type" gorm:"type:varchar(10);not null"`
	TotalPrice    float64 `json:"total_price" gorm:"type:numeric(10,2);not null"` // (purchase_price-discount)*purchase_qty_boxes
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
