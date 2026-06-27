package dto

type CreatePrescriptionRequest struct {
	MedicineID     string          `json:"medicine_id"`
	AppointmentID  string          `json:"appointment_id"`
	PatientID      string          `json:"patient_id"`
	PrescribedBy   string          `json:"prescribed_by"`
	MedicineArray  []MedicineArray `json:"medicine_array"`
	OrganisationID string          `json:"organisation_id"`
}
type MedicineArray struct {
	MedicineID      string  `json:"medicine_id"`
	MedicineName    string  `json:"medicine_name"`
	DurationDay     float64 `json:"duration_day"`
	DurationType    string  `json:"duration_type"`
	Quantity        int     `json:"quantity"`
	MedicineType    string  `json:"medicine_type"`
	FoodInstruction string  `json:"food_instruction"`
	Morning         float64 `json:"morning"`
	Afternoon       float64 `json:"afternoon"`
	Night           float64 `json:"night"`
	Dosage          string  `json:"dosage"`
}
type FindManyRequest struct {
	OrganisationID string `json:"organisation_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
}
type UpdateRequest struct {
	PrescriptionID string          `json:"prescription_id"`
	AppointmentID  string          `json:"appointment_id"`
	MedicineArr    []MedicineArray `json:"medicine_array"`
	UserID         string          `json:"user_id"`
}
type PresPatients struct {
	PatientID      string  `json:"patient_id"`
	OrganisationID string  `json:"organisation_id"`
	Limit          float64 `json:"limit"`
	Pageno         float64 `json:"page_no"`
}
type DispensePayload struct {
	PrescriptionID string             `json:"prescription_id" binding:"required,uuid"`
	SupplierID     string             `json:"supplier_id" binding:"required,uuid"`
	OrganisationID string             `json:"organisation_id" binding:"required,uuid"`
	PatientID      string             `json:"patient_id" binding:"required,uuid"`
	CashierID      string             `json:"cashier_id" binding:"required,uuid"`
	PaymentMode    string             `json:"payment_mode" binding:"required,oneof=CASH UPI CARD"`
	Financials     FinancialsDTO      `json:"financials" binding:"required"`
	DispensedItems []DispensedItemDTO `json:"dispensed_items" binding:"required,div=0"`
}

// FinancialsDTO isolates the customer-facing receipt totals
type FinancialsDTO struct {
	SubtotalAmount  float64 `json:"subtotal_amount"`
	TaxAmount       float64 `json:"tax_amount"`
	DiscountAmount  float64 `json:"discount_amount"`
	TotalAmountPaid float64 `json:"total_amount_paid"`
}

// DispensedItemDTO maps each physical batch item being sold
type DispensedItemDTO struct {
	MedicineID        string  `json:"medicine_id" binding:"required,uuid"`
	BatchID           string  `json:"batch_id" binding:"required,uuid"`
	QuantitySoldUnits int     `json:"quantity_sold_units" binding:"required,gt=0"`
	UnitPriceCharged  float64 `json:"unit_price_charged" binding:"required,gt=0"`
	ComputedItemTotal float64 `json:"computed_item_total"`
}
