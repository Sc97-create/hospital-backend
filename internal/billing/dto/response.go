package dto

import "hospital-backend/pkg/types"

type MedInvoiceItemResponse struct {
	PrescriptionItemID            string        `json:"prescription_item_id"`
	PrescriptionID                string        `json:"prescription_id"`     // from invoices.prescription_id
	MedicineInventoryID           string        `json:"medicine_inventory_id"`
	OrganisationID                string        `json:"organisation_id"`
	MedicineID                    string        `json:"medicine_id"`
	CurrentStockUnit              int64         `json:"current_stock_unit"`
	DispensedQty                  int64         `json:"dispensed_qty"`
	CurrentStockUnitAfterDispense int64         `json:"current_stock_unit_after_dispense"`
	Pricing                       types.Pricing `json:"pricing"`
	CashierID                     string        `json:"cashier_id"`
	PrescribedQty                 int64         `json:"prescribed_qty"`      // from prescription_items.quantity
	AlreadyDispensed              int64         `json:"already_dispensed"`   // from prescription_items.balance_after_dispense
}
