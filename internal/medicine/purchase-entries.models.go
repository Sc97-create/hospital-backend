package medicine

import "time"

type MPurchaseEntry struct {
	ID             string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	InvoiceNumber  string    `json:"invoice_number" gorm:"type:varchar(100);not null"`
	InvoiceDate    time.Time `json:"invoice_date" gorm:"type:date;not null"` // FIXED: Changed gorm tag from varchar to date
	SupplierID     string    `json:"supplier_id" gorm:"type:uuid;not null"`
	PaymentDueDate time.Time `json:"payment_due_date" gorm:"type:date;"` // FIXED: Changed gorm tag from varchar to date
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}
