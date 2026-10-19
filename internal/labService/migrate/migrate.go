package migrate

import (
	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func Migrate(database db.Postgre) error {
	if err := database.AutoMigrate(&diagnosticcentre.DiagnosticCentre{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&labservice.LabTest{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&labservice.LabOrders{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&labservice.LabTestTemplate{}); err != nil {
		return err
	}
	return ensureLabForeignKeys(database.GormDriver)
}

func ensureLabForeignKeys(gdb *gorm.DB) error {
	drops := []string{
		`ALTER TABLE lab_orders DROP CONSTRAINT IF EXISTS fk_lab_orders_appointment`,
	}
	for _, sql := range drops {
		if err := gdb.Exec(sql).Error; err != nil {
			return err
		}
	}
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE diagnostic_centres
		   ADD CONSTRAINT fk_diagnostic_centres_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE lab_tests
		   ADD CONSTRAINT fk_lab_tests_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE lab_orders
		   ADD CONSTRAINT fk_lab_orders_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE lab_orders
		   ADD CONSTRAINT fk_lab_orders_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE lab_orders
		   ADD CONSTRAINT fk_lab_orders_doctor
		   FOREIGN KEY (doctor_id) REFERENCES users(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE lab_orders
		   ADD CONSTRAINT fk_lab_orders_diagnostic_centre
		   FOREIGN KEY (diagnostic_centre_id) REFERENCES diagnostic_centres(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE lab_test_templates
		   ADD CONSTRAINT fk_lab_test_templates_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
