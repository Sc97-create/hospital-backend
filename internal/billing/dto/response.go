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

// InvoiceByPrescriptionResponse is also reused by the by-appointment lookup
// (GetInvoiceByAppointmentID) — same invoice shape either way.
type InvoiceByPrescriptionResponse struct {
	ID             string  `json:"id"`
	InvoiceCode    string  `json:"invoice_code"`
	PaymentType    string  `json:"payment_type"`
	PrescriptionID string  `json:"prescription_id,omitempty"`
	AppointmentID  string  `json:"appointment_id,omitempty"`
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

// BillDetailsByPrescriptionResponse is the combined patient + appointment + fee view
// for a prescription (consultation + prescription invoice lines when present).
type BillDetailsByPrescriptionResponse struct {
	PatientDetail     PatientDetail     `json:"patientDetail"`
	AppointmentDetail AppointmentDetail `json:"appointmentDetail"`
	PaymentDetails    PaymentDetails    `json:"paymentDetails"`
	InvoiceStatus     string            `json:"invoice_status"`
	InvoiceCode       string            `json:"invoice_code"`
	CreatedAt         string            `json:"created_at"`
	InvoiceID         string            `json:"invoice_id"`
}

type PatientDetail struct {
	ID     string `json:"id"`
	UHID   string `json:"uhid"`
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Gender string `json:"gender"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
}

type AppointmentDetail struct {
	ID              string `json:"id"`
	AppointmentCode string `json:"appointment_code"`
	VisitType       string `json:"visit_type"`
	Status          string `json:"status"`
}

// PaymentDetails holds fee lines for consultation and/or prescription invoices.
// Grand total is left to the caller.
type PaymentDetails struct {
	Consultation *FeeLine `json:"consultation,omitempty"`
	Prescription *FeeLine `json:"prescription,omitempty"`
}

type FeeLine struct {
	Code          string  `json:"code"`
	Category      string  `json:"category,omitempty"`
	Qty           int     `json:"qty"`
	Tax           float64 `json:"tax"`
	Discount      float64 `json:"discount"`
	TotalAmount   float64 `json:"total_amount"`
	InvoiceStatus string  `json:"invoice_status"`
	PaymentMode   string  `json:"payment_mode"`
}
