package dto

type CheckoutReq struct {
	PrescriptionID string          `json:"prescription_id"`
	OrganisationID string          `json:"organisation_id"`
	PatientID      string          `json:"patient_id"`
	CashierID      string          `json:"cashier_id"`
	SupplierID     string          `json:"supplier_id"`
	PaymentMode    string          `json:"payment_mode"`
	Financials     Financial       `json:"financials"`
	DispensedItems []DispensedItem `json:"dispensed_items"`
}
type DispensedItem struct {
	MedicineID          string  `json:"medicine_id"`
	MedicineInventoryID string  `json:"medicine_inventory_id"`
	PrescriptionItemID  string  `json:"prescription_item_id"`
	BatchNo             string  `json:"batch_no"`
	CurrentStockUnits   float64 `json:"current_stock_units"`
	QuantitySoldUnits   float64 `json:"quantity_sold_units"`
	UnitPriceCharged    float64 `json:"unit_price_charged"`
	ComputedItemTotal   float64 `json:"computed_item_total"` //subtotal
	TotalAmount         float64 `json:"total_amount"`        //totalprice
}
type Financial struct {
	SubtotalAmount float64 `json:"sub_total_amount"`
	TaxAmount      float64 `json:"tax_amount"`
	DiscountAmount float64 `json:"discount_amount"`
	TotalAmount    float64 `json:"total_amount"`
}
