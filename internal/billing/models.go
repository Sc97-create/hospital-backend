package billing

import "time"

type Invoice struct {
	ID          string      `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceCode string      `json:"invoice_code" gorm:"type:text;not null"`
	PaymentType PaymentType `json:"payment_type" gorm:"column:payment_type;type:varchar(30);not null;default:prescription"`
	// BillingID is the source entity id: prescription id, appointment id, lab order id, or subscription id.
	// Uniqueness of (payment_type, billing_id) is a partial index in Migrate; subscription_renewal may repeat.
	BillingID string `json:"billing_id" gorm:"column:billing_id;type:uuid;not null"`
	// PatientID is required for hospital invoices. Subscription invoices leave it empty and set CustomerID.
	PatientID string `json:"patient_id" gorm:"type:uuid"`
	// CustomerID is required for subscription_new and subscription_renewal.
	CustomerID     *string   `json:"customer_id,omitempty" gorm:"type:uuid"`
	Status         string    `json:"status" gorm:"default:unpaid"`
	CashierID      string    `json:"cashier_id" gorm:"type:uuid"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	SubtotalAmount float64   `json:"sub_total_amount" gorm:"type:numeric(10,2);not null"`
	TaxAmount      float64   `json:"tax_amount" gorm:"type:numeric(10,2);not null"`
	TotalAmount    float64   `json:"total_amount" gorm:"type:numeric(10,2);not null"`
	DiscountAmount float64   `json:"discount_amount" gorm:"type:numeric(10,2);not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoCreateTime"`
}

type TodayInvoiceCollectionRow struct {
	PaymentMode string  `gorm:"column:payment_mode"`
	Count       int     `gorm:"column:invoice_count"`
	Amount      float64 `gorm:"column:total_amount"`
}

type InvoiceItem struct {
	ID                  string    `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceID           string    `json:"invoice_id" gorm:"type:uuid;not null"`
	MedicineID          string    `json:"medicine_id" gorm:"type:uuid;not null"`
	MedicineInventoryID string    `json:"medicine_inventory_id" gorm:"type:uuid;not null"`
	PrescriptionItemID  string    `json:"prescription_item_id" gorm:"type:uuid"`
	BatchNo             string    `json:"batch_no" gorm:"not null"`
	SubtotalPrice       float64   `json:"sub_total_price" gorm:"numeric(10,2);not null"`
	TotalPrice          float64   `json:"total_price" gorm:"numeric(10,2);not null"`
	DispensedQty        int       `json:"dispensed_qty" gorm:"default:0;"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// BillDetailsRow is the flat scan target for GetBillDetailsByPrescriptionID raw SQL.
// Nullable LEFT JOIN columns use pointers so missing consultation/prescription invoices scan cleanly.
type BillDetailsRow struct {
	PrescriptionID   string `gorm:"column:prescription_id"`
	PrescriptionCode string `gorm:"column:prescription_code"`

	PatientID     *string `gorm:"column:patient_id"`
	PatientUHID   *string `gorm:"column:patient_uhid"`
	PatientName   *string `gorm:"column:patient_name"`
	PatientAge    *int    `gorm:"column:patient_age"`
	PatientGender *string `gorm:"column:patient_gender"`
	PatientPhone  *string `gorm:"column:patient_phone"`
	PatientEmail  *string `gorm:"column:patient_email"`

	AppointmentID     *string `gorm:"column:appointment_id"`
	AppointmentCode   *string `gorm:"column:appointment_code"`
	VisitType         *string `gorm:"column:visit_type"`
	AppointmentStatus *string `gorm:"column:appointment_status"`

	ConsultationQty           *int     `gorm:"column:consultation_qty"`
	ConsultationTax           *float64 `gorm:"column:consultation_tax"`
	ConsultationDiscount      *float64 `gorm:"column:consultation_discount"`
	ConsultationTotalAmount   *float64 `gorm:"column:consultation_total_amount"`
	ConsultationInvoiceStatus *string  `gorm:"column:consultation_invoice_status"`
	ConsultationPaymentMode   *string  `gorm:"column:consultation_payment_mode"`

	PrescriptionQty              *int       `gorm:"column:prescription_qty"`
	PrescriptionTax              *float64   `gorm:"column:prescription_tax"`
	PrescriptionDiscount         *float64   `gorm:"column:prescription_discount"`
	PrescriptionTotalAmount      *float64   `gorm:"column:prescription_total_amount"`
	PrescriptionInvoiceStatus    *string    `gorm:"column:prescription_invoice_status"`
	PrescriptionPaymentMode      *string    `gorm:"column:prescription_payment_mode"`
	PrescriptionInvoiceID        *string    `gorm:"column:prescription_invoice_id"`
	PrescriptionInvoiceCode      *string    `gorm:"column:prescription_invoice_code"`
	PrescriptionInvoiceCreatedAt *time.Time `gorm:"column:prescription_invoice_created_at"`
}
