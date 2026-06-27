package medicine

import "time"

type MPurchaseEntry struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	InvoiceNumber  string    `json:"invoice_number" gorm:"type:varchar(100);not null"`
	InvoiceDate    time.Time `json:"invoice_date" gorm:"type:varchar(100);not null"`
	SupplierID     string    `json:"supplier_id" gorm:"type:uuid;not null"`
	PaymentDueDate time.Time `json:"payment_due_date" gorm:"type:varchar(100);"`
}
