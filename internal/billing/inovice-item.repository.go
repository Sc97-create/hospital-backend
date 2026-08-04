package billing

import (
	"hospital-backend/internal/billing/dto"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InvoiceItemRepo interface {
	Create(log *zap.Logger, tx *gorm.DB, InvItem []InvoiceItem) error
	GetInvoiceItemsByInvoiceID(log *zap.Logger, query string, args ...interface{}) ([]dto.MedInvoiceItemResponse, error)
}

func (d *DB) Create(log *zap.Logger, tx *gorm.DB, InvItem []InvoiceItem) error {
	log = ensureLog(log)
	err := tx.CreateInBatches(&InvItem, len(InvItem)).Error
	if err != nil {
		log.Error("billing repo error", zap.String("op", "CreateInvoiceItems"), zap.Error(err))
		return err
	}
	return nil
}

func (d *DB) GetInvoiceItemsByInvoiceID(log *zap.Logger, query string, args ...interface{}) ([]dto.MedInvoiceItemResponse, error) {
	log = ensureLog(log)
	var invoiceItems []dto.MedInvoiceItemResponse
	err := d.db.Raw(query, args...).Scan(&invoiceItems).Error
	if err != nil {
		log.Error("billing repo error", zap.String("op", "GetInvoiceItemsByInvoiceID"), zap.Error(err))
		return nil, err
	}
	return invoiceItems, nil
}
