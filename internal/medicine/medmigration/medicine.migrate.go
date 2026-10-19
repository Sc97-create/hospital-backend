package medmigration

import (
	"hospital-backend/internal/medicine"
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func Automigrate(database db.Postgre) error {
	// Parent tables first so FK constraints can be applied safely.
	if err := database.AutoMigrate(&medicine.Medicine{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&medicine.Supplier{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&medicine.MPurchaseEntry{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&medicine.MedicineInventory{}); err != nil {
		return err
	}
	return ensureMedicineForeignKeys(database.GormDriver)
}

func ensureMedicineForeignKeys(gdb *gorm.DB) error {
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE medicines
		   ADD CONSTRAINT fk_medicines_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE suppliers
		   ADD CONSTRAINT fk_suppliers_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE m_purchase_entries
		   ADD CONSTRAINT fk_m_purchase_entries_supplier
		   FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE m_purchase_entries
		   ADD CONSTRAINT fk_m_purchase_entries_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE medicine_inventories
		   ADD CONSTRAINT fk_medicine_inventories_medicine
		   FOREIGN KEY (medicine_id) REFERENCES medicines(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE medicine_inventories
		   ADD CONSTRAINT fk_medicine_inventories_supplier
		   FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE medicine_inventories
		   ADD CONSTRAINT fk_medicine_inventories_purchase_entry
		   FOREIGN KEY (purchase_entry_id) REFERENCES m_purchase_entries(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE medicine_inventories
		   ADD CONSTRAINT fk_medicine_inventories_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
