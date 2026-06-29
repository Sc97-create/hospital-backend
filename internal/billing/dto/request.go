package dto

type CheckoutReq struct {
	PrescriptionID string    `json:"prescription_id"`
	OrganisationID string    `json:"organisation_id"`
	PatientID      string    `json:"patient_id"`
	CashierID      string    `json:"cashier_id"`
	SupplierID     string    `json:"supplier_id"`
	PaymentMode    string    `json:"payment_mode"`
	Financials     Financial `json:"financials"`
	DispensedItems []DispensedItem
}
type DispensedItem struct {
	MedicineID        string  `json:"medicine_id"`
	BatchNo           string  `json:"batch_no"`
	QuantitySoldUnits int64   `json:"quantity_sold_units"`
	UnitPriceCharged  float64 `json:"unit_price_charged"`
	ComputedItemTotal float64 `json:"computed_item_total"` //subtotal
	TotalAmount       float64 `json:"total_amount"`        //totalprice
}
type Financial struct {
	SubtotalAmount float64 `json:"sub_total_amount"`
	TaxAmount      float64 `json:"tax_amount"`
	DiscountAmount float64 `json:"discount_amount"`
	TotalAmount    float64 `json:"total_amount"`
}
