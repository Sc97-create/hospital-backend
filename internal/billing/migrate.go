package billing

import (
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func Migrate(database db.Postgre) error {
	if err := database.AutoMigrate(&Invoice{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&InvoiceItem{}); err != nil {
		return err
	}
	return ensureBillingForeignKeys(database.GormDriver)
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
