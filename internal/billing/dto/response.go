package dto

import "hospital-backend/pkg/types"

type MedInvoiceItemResponse struct {
	PrescriptionItemID            string        `json:"prescription_item_id"`
	PrescriptionID                string        `json:"prescription_id"` // from invoices.prescription_id
	MedicineInventoryID           string        `json:"medicine_inventory_id"`
	OrganisationID                string        `json:"organisation_id"`
	MedicineID                    string        `json:"medicine_id"`
	CurrentStockUnit              int64         `json:"current_stock_unit"`
	DispensedQty                  int64         `json:"dispensed_qty"`
	CurrentStockUnitAfterDispense int64         `json:"current_stock_unit_after_dispense"`
	Pricing                       types.Pricing `json:"pricing"`
	CashierID                     string        `json:"cashier_id"`
	PrescribedQty                 int64         `json:"prescribed_qty"`         // from prescription_items.quantity
	BalanceAfterDispense          int64         `json:"balance_after_dispense"` // remaining qty after dispenses
	PrescriptionItemStatus        string        `json:"prescription_item_status"`
}
type InvoiceResponse struct {
	InvoiceID  string `json:"invoice_id"`
	PaymentURL string `json:"payment_url"`
}

type InvoiceByPrescriptionResponse struct {
	ID             string  `json:"id"`
	InvoiceCode    string  `json:"invoice_code"`
	PrescriptionID string  `json:"prescription_id"`
	PatientID      string  `json:"patient_id"`
	Status         string  `json:"status"`
	CashierID      string  `json:"cashier_id"`
	OrganisationID string  `json:"organisation_id"`
	PaymentMode    string  `json:"payment_mode"`
	SubtotalAmount float64 `json:"sub_total_amount"`
	TaxAmount      float64 `json:"tax_amount"`
	TotalAmount    float64 `json:"total_amount"`
	DiscountAmount float64 `json:"discount_amount"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}
