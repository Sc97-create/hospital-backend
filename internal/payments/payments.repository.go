package payments

import "gorm.io/gorm"

type DB struct {
	db *gorm.DB
}

func NewPaymentsDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

type IPaymentsRepository interface {
	Create(db *gorm.DB, payment Payments) (err error)
	FindInvoiceByPaymentAttempt(query string, args ...interface{}) (Payments, error)
	FindByInvoiceID(invoiceID string) (Payments, error)
}

func (c *DB) Create(db *gorm.DB, payments Payments) (err error) {
	return db.Create(&payments).Error
}
func (c *DB) FindInvoiceByPaymentAttempt(query string, args ...interface{}) (Payments, error) {
	var payment Payments
	err := c.db.Raw(query, args...).Scan(&payment).Error
	return payment, err
}
func (c *DB) FindByInvoiceID(invoiceID string) (Payments, error) {
	var payment Payments
	err := c.db.Where("invoice_id = ?", invoiceID).Order("created_at DESC").First(&payment).Error
	return payment, err
}
