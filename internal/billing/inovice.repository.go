package billing

import (
	"fmt"

	"gorm.io/gorm"
)

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
	query := tx.Model(&Invoice{}).Where("id = ?", invoiceID)
	// paid transitions only from unpaid — prevents double confirm / double dispense
	if status == StatusPaid {
		query = query.Where("status = ?", StatusUnpaid)
	}
	result := query.Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if status == StatusPaid && result.RowsAffected == 0 {
		return fmt.Errorf("invoice %s is already paid or not found", invoiceID)
	}
	return nil
}
