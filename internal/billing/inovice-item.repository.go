package billing

import (
	"hospital-backend/internal/billing/dto"

	"gorm.io/gorm"
)

type InvoiceItemRepo interface {
	Create(tx *gorm.DB, InvItem []InvoiceItem) error
	GetInvoiceItemsByInvoiceID(query string, args ...interface{}) ([]dto.MedInvoiceItemResponse, error)
}

func (d *DB) Create(tx *gorm.DB, InvItem []InvoiceItem) error {
	return tx.CreateInBatches(&InvItem, len(InvItem)).Error
}
func (d *DB) GetInvoiceItemsByInvoiceID(query string, args ...interface{}) ([]dto.MedInvoiceItemResponse, error) {
	var invoiceItems []dto.MedInvoiceItemResponse
	err := d.db.Raw(query, args...).Scan(&invoiceItems).Error
	return invoiceItems, err
}
