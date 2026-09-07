package billing

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InvoiceWithPayment struct {
	Invoice
	PaymentMode string `gorm:"column:payment_mode"`
}

type InvoiceRepo interface {
	CreateInvoice(log *zap.Logger, tx *gorm.DB, Inv Invoice) error
	UpdateInvoiceStatus(log *zap.Logger, tx *gorm.DB, invoiceID string, status string) error
	GetInvoiceByPrescriptionID(log *zap.Logger, query string, args ...any) (InvoiceWithPayment, error)
	GetInvoiceByAppointmentID(log *zap.Logger, query string, args ...any) (InvoiceWithPayment, error)
	GetInvoiceByID(log *zap.Logger, invoiceID string) (Invoice, error)
	GetBillDetailsByPrescriptionID(log *zap.Logger, query string, args ...any) (BillDetailsRow, error)
}

type DB struct {
	db *gorm.DB
}

func NewDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

func (d *DB) CreateInvoice(log *zap.Logger, tx *gorm.DB, Inv Invoice) error {
	log = ensureLog(log)
	err := tx.Create(&Inv).Error
	if err != nil {
		log.Error("billing repo error", zap.String("op", "CreateInvoice"), zap.Error(err))
		return err
	}
	return nil
}

func (d *DB) GetInvoiceByPrescriptionID(log *zap.Logger, query string, args ...any) (InvoiceWithPayment, error) {
	log = ensureLog(log)
	var row InvoiceWithPayment
	err := d.db.Raw(query, args...).Scan(&row).Error
	if err != nil {
		log.Error("billing repo error", zap.String("op", "GetInvoiceByPrescriptionID"), zap.Error(err))
		return row, err
	}
	if row.ID == "" {
		return row, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (d *DB) GetInvoiceByAppointmentID(log *zap.Logger, query string, args ...any) (InvoiceWithPayment, error) {
	log = ensureLog(log)
	var row InvoiceWithPayment
	err := d.db.Raw(query, args...).Scan(&row).Error
	if err != nil {
		log.Error("billing repo error", zap.String("op", "GetInvoiceByAppointmentID"), zap.Error(err))
		return row, err
	}
	if row.ID == "" {
		return row, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (d *DB) GetInvoiceByID(log *zap.Logger, invoiceID string) (Invoice, error) {
	log = ensureLog(log)
	var invoice Invoice
	err := d.db.Where("id = ?", invoiceID).First(&invoice).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("billing repo error", zap.String("op", "GetInvoiceByID"), zap.Error(err))
		}
		return invoice, err
	}
	return invoice, err
}

func (d *DB) GetBillDetailsByPrescriptionID(log *zap.Logger, query string, args ...any) (BillDetailsRow, error) {
	log = ensureLog(log)
	var row BillDetailsRow
	err := d.db.Raw(query, args...).Scan(&row).Error
	if err != nil {
		log.Error("billing repo error", zap.String("op", "GetBillDetailsByPrescriptionID"), zap.Error(err))
		return row, err
	}
	if row.PrescriptionID == "" {
		return row, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (d *DB) UpdateInvoiceStatus(log *zap.Logger, tx *gorm.DB, invoiceID string, status string) error {
	log = ensureLog(log)
	query := tx.Model(&Invoice{}).Where("id = ?", invoiceID)
	// paid transitions only from unpaid — prevents double confirm / double dispense
	if status == StatusPaid {
		query = query.Where("status = ?", StatusUnpaid)
	}
	result := query.Update("status", status)
	if result.Error != nil {
		log.Error("billing repo error", zap.String("op", "UpdateInvoiceStatus"), zap.Error(result.Error))
		return result.Error
	}
	if status == StatusPaid && result.RowsAffected == 0 {
		err := fmt.Errorf("invoice %s is already paid or not found", invoiceID)
		log.Warn("invoice status update failed",
			zap.String("invoice_id", invoiceID),
			zap.String("status", status),
			zap.String("reason", "already_paid"),
		)
		return err
	}
	if status == StatusPaid {
		log.Info("invoice status update success",
			zap.String("invoice_id", invoiceID),
			zap.String("status", status),
		)
	}
	return nil
}
