package payments

import (
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func Migrate(database db.Postgre) error {
	if err := database.AutoMigrate(&Payments{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&PaymentAttempts{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&Refunds{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&WebhookEvents{}); err != nil {
		return err
	}
	return ensurePaymentsForeignKeys(database.GormDriver)
}

func ensurePaymentsForeignKeys(gdb *gorm.DB) error {
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE payment_attempts
		   ADD CONSTRAINT fk_payment_attempts_payment
		   FOREIGN KEY (payment_id) REFERENCES payments(id)
		   ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE refunds
		   ADD CONSTRAINT fk_refunds_payment_attempt
		   FOREIGN KEY (payment_attempt_id) REFERENCES payment_attempts(id)
		   ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE payments
		   ADD CONSTRAINT fk_payments_invoice
		   FOREIGN KEY (invoice_id) REFERENCES invoices(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE payments
		   ADD CONSTRAINT fk_payments_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
