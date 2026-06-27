package dto

import "time"

type Supplier struct {
	UserID            string  `json:"user_id"`
	OrganisationID    string  `json:"organisation_id"`
	Name              string  `json:"name"`
	PaymentTerms      string  `json:"payment_terms"`
	EmailID           string  `json:"email_id"`
	DrugLicenseNumber string  `json:"drug_license_number"`
	ContactNumber     string  `json:"contact_number"`
	CreditLimit       float64 `json:"credit_limit"`
	GstNumber         string  `json:"gst_number"`
}

type RequestPayload struct {
	UserID         string         `json:"user_id"`
	SupplierID     string         `json:"supplier_id"`
	OrganisationID string         `json:"organisation_id"`
	InvoiceNo      string         `json:"invoice_no"`
	InvoiceDate    time.Time      `json:"invoice_date"`
	PaymentDueDate string         `json:"payment_due_date"`
	MedicineArray  []MedicineInfo `json:"medicine_array"`
	BatchNumber    string         `json:"batch_number"`
}
type MedicineInfo struct {
	MedInventoryID   string  `json:"med_inventory_id"`
	MedicineID       string  `json:"medicine_id"`
	Name             string  `json:"name"`
	Form             string  `json:"form"`
	Strength         string  `json:"strength"`
	BatchNumber      string  `json:"batch_number"`
	ExpiryDate       string  `json:"expiry_date"`
	Quantity         float64 `json:"quantity"`
	MRP              float64 `json:"mrp"`
	Discount         float64 `json:"discount"`
	PurchasePrice    float64 `json:"purchase_price"`
	SellingPrice     float64 `json:"selling_price"`
	HsnCode          string  `json:"hsn_code"`
	PurchaseQtyBoxes int     `json:"purchase_qty_box"`
	UnitPerBoxes     int     `json:"units_per_box"`
	ShelfLocation    string  `json:"shelf_location"`
	Add              bool    `json:"add"`
	ReorderLevel     int     `json:"reorder_level"`
	MaxStockTarget   int     `json:"max_stock_target"`
}
