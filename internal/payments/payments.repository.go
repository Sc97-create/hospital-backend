package payments

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IPaymentsRepository interface {
	Create(log *zap.Logger, db *gorm.DB, payment Payments) (err error)
	UpdateSource(log *zap.Logger, tx *gorm.DB, paymentID, source, channel string) error
	FindInvoiceByPaymentAttempt(log *zap.Logger, query string, args ...interface{}) (Payments, error)
	FindByInvoiceID(log *zap.Logger, invoiceID string) (Payments, error)
	FindByIdempotencyKey(log *zap.Logger, idempotencyKey string) (Payments, error)
}

type DB struct {
	db *gorm.DB
}

func NewPaymentsDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

func (c *DB) Create(log *zap.Logger, db *gorm.DB, payments Payments) (err error) {
	log = ensureLog(log)
	err = db.Create(&payments).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (c *DB) UpdateSource(log *zap.Logger, tx *gorm.DB, paymentID, source, channel string) error {
	log = ensureLog(log)
	err := tx.Model(&Payments{}).Where("id = ?", paymentID).Updates(map[string]interface{}{
		"source":  source,
		"channel": channel,
	}).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "UpdateSource"), zap.Error(err))
		return err
	}
	return nil
}

func (c *DB) FindInvoiceByPaymentAttempt(log *zap.Logger, query string, args ...interface{}) (Payments, error) {
	log = ensureLog(log)
	var payment Payments
	err := c.db.Raw(query, args...).Scan(&payment).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "FindInvoiceByPaymentAttempt"), zap.Error(err))
		return payment, err
	}
	return payment, nil
}

func (c *DB) FindByInvoiceID(log *zap.Logger, invoiceID string) (Payments, error) {
	log = ensureLog(log)
	var payment Payments
	err := c.db.Where("invoice_id = ?", invoiceID).Order("created_at DESC").First(&payment).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("payments repo error", zap.String("op", "FindByInvoiceID"), zap.Error(err))
		}
		return payment, err
	}
	return payment, nil
}

func (c *DB) FindByIdempotencyKey(log *zap.Logger, idempotencyKey string) (Payments, error) {
	log = ensureLog(log)
	var payment Payments
	err := c.db.Where("idempotency_key = ?", idempotencyKey).First(&payment).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("payments repo error", zap.String("op", "FindByIdempotencyKey"), zap.Error(err))
		}
		return payment, err
	}
	return payment, nil
}
