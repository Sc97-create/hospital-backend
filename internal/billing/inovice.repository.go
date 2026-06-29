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
}

func (d *DB) CreateInvoice(tx *gorm.DB, Inv Invoice) error {
	return tx.Create(&Inv).Error
}
