package billing

import "gorm.io/gorm"

type DB struct {
	db *gorm.DB
}

func NewDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

type InvoiceRepo interface {
	CreateInvoice(tx *gorm.DB, Inv Invoice) error
	UpdateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error
}

func (d *DB) CreateInvoice(tx *gorm.DB, Inv Invoice) error {
	return tx.Create(&Inv).Error
}
func (d *DB) UpdateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error {
	return tx.Model(&Invoice{}).Where("id = ?", invoiceID).Update("status", status).Error
}
