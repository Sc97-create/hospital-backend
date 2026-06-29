package billing

import "gorm.io/gorm"

type InvoiceItemRepo interface {
	Create(tx *gorm.DB, InvItem []InvoiceItem) error
}

func (d *DB) Create(tx *gorm.DB, InvItem []InvoiceItem) error {
	return tx.CreateInBatches(&InvItem, len(InvItem)).Error
}
