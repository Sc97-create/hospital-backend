package medicine

import (
	"hospital-backend/pkg/types"
	"time"
)

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
	Pricing types.Pricing `json:"medicine_pricing" gorm:"type:jsonb;not null"`
}
type MixedMedInventory struct {
	ID                string        `json:"id"`
	MedicineID        string        `json:"medicine_id"`
	BatchNo           string        `json:"batch_no"`
	ExpiresAt         time.Time     `json:"expiry_at"`
	MedicineName      string        `json:"medicine_name"`
	MedForm           string        `json:"med_form"`
	MedicineStrength  string        `json:"medicine_strength"`
	ShelfLocation     string        `json:"shelf_location"`
	PurchaseQtyBoxes  int           `json:"purchase_qty_boxes"`
	UnitsPerBox       int           `json:"units_per_box"`
	CurrentStockUnits int           `json:"current_stock_units"`
	Pricing           types.Pricing `json:"medicine_pricing"`
}
