package billing

import "time"

type Invoice struct {
	ID             string    `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceCode    string    `json:"invoice_code" gorm:"type:text;not null"`
	PrescriptionID string    `json:"prescription_id" gorm:"type:uuid"`
	PatientID      string    `json:"patient_id" gorm:"type:uuid;not null"`
	Status         string    `json:"status" gorm:"default:unpaid"`
	CashierID      string    `json:"cashier_id" gorm:"type:uuid;not null"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	SubtotalAmount float64   `json:"sub_total_amount" gorm:"type:numeric(10,2);not null"`
	TaxAmount      float64   `json:"tax_amount" gorm:"type:numeric(10,2);not null"`
	TotalAmount    float64   `json:"total_amount" gorm:"type:numeric(10,2);not null"`
	DiscountAmount float64   `json:"discount_amount" gorm:"type:numeric(10,2);not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoCreateTime"`
}
type InvoiceItem struct {
	ID            string    `json:"id" gorm:"type:uuid;not null;primaryKey"`
	InvoiceID     string    `json:"invoice_id" gorm:"type:uuid; not null"`
	MedicineID    string    `json:"medicine_id" gorm:"type:uuid;not null"`
	BatchNo       string    `json:"batch_no" gorm:"not null"`
	SubtotalPrice float64   `json:"sub_total_price" gorm:"numeric(10,2);not null"`
	TotalPrice    float64   `json:"total_price" gorm:"numeric(10,2);not null"`
	DispensedQty  int       `json:"dispensed_qty" gorm:"default:0;"`
	Pendingqty    int       `json:"pending_qty" gorm:"default:0;"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}
