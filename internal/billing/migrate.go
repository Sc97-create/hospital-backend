package billing

import (
	"strings"

	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func Migrate(database db.Postgre) error {
	if err := database.AutoMigrate(&Invoice{}); err != nil {
		return err
	}
	if err := ensureInvoiceBillingUniqueIndex(database.GormDriver); err != nil {
		return err
	}
	if err := ensureInvoicePartyColumns(database.GormDriver); err != nil {
		return err
	}
	if err := database.AutoMigrate(&InvoiceItem{}); err != nil {
		return err
	}
	return ensureBillingForeignKeys(database.GormDriver)
}

// ensureInvoiceBillingUniqueIndex keeps one invoice per billing_id for every payment_type
// except subscription_renewal, which can repeat for the same subscription.
// CREATE INDEX IF NOT EXISTS would leave an older full unique index in place, so an
// existing index is replaced only when its definition does not already exclude renewal.
func ensureInvoiceBillingUniqueIndex(gdb *gorm.DB) error {
	var indexDef string
	err := gdb.Raw(`
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND indexname = 'idx_payment_type_billing_id'
	`).Scan(&indexDef).Error
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(indexDef), "subscription_renewal") {
		return nil
	}
	if err := gdb.Exec(`DROP INDEX IF EXISTS idx_payment_type_billing_id`).Error; err != nil {
		return err
	}
	return gdb.Exec(`
		CREATE UNIQUE INDEX idx_payment_type_billing_id
		ON invoices (payment_type, billing_id)
		WHERE payment_type <> 'subscription_renewal'
	`).Error
}

// ensureInvoicePartyColumns lets subscription invoices omit patient and cashier.
// Hospital invoices still require a patient in the checkout service.
// A subscription invoice must have a customer; every other type must have a patient.
func ensureInvoicePartyColumns(gdb *gorm.DB) error {
	if err := gdb.Exec(`ALTER TABLE invoices ALTER COLUMN patient_id DROP NOT NULL`).Error; err != nil {
		return err
	}
	if err := gdb.Exec(`ALTER TABLE invoices ALTER COLUMN cashier_id DROP NOT NULL`).Error; err != nil {
		return err
	}
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE invoices DROP CONSTRAINT IF EXISTS chk_invoices_party`,
		`ALTER TABLE invoices ADD CONSTRAINT chk_invoices_party CHECK (
			(payment_type IN ('subscription_new', 'subscription_renewal') AND customer_id IS NOT NULL)
			OR
			(payment_type NOT IN ('subscription_new', 'subscription_renewal') AND patient_id IS NOT NULL)
		)`,
		`ALTER TABLE invoices
		   ADD CONSTRAINT fk_invoices_customer
		   FOREIGN KEY (customer_id) REFERENCES customers(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}

func ensureBillingForeignKeys(gdb *gorm.DB) error {
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE invoice_items
		   ADD CONSTRAINT fk_invoice_items_invoice
		   FOREIGN KEY (invoice_id) REFERENCES invoices(id)
		   ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE invoice_items
		   ADD CONSTRAINT fk_invoice_items_medicine
		   FOREIGN KEY (medicine_id) REFERENCES medicines(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE invoice_items
		   ADD CONSTRAINT fk_invoice_items_medicine_inventory
		   FOREIGN KEY (medicine_inventory_id) REFERENCES medicine_inventories(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE invoice_items
		   ADD CONSTRAINT fk_invoice_items_prescription_item
		   FOREIGN KEY (prescription_item_id) REFERENCES prescription_items(id)
		   ON UPDATE CASCADE ON DELETE SET NULL`,
		`ALTER TABLE invoices
		   ADD CONSTRAINT fk_invoices_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE invoices
		   ADD CONSTRAINT fk_invoices_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE invoices
		   ADD CONSTRAINT fk_invoices_cashier
		   FOREIGN KEY (cashier_id) REFERENCES users(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
