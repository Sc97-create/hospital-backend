package appointments

import (
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func AutoMigrate(database db.Postgre) error {
	if err := database.AutoMigrate(&Appointment{}); err != nil {
		return err
	}
	return ensureAppointmentForeignKeys(database.GormDriver)
}

func ensureAppointmentForeignKeys(gdb *gorm.DB) error {
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE appointments
		   ADD CONSTRAINT fk_appointments_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE appointments
		   ADD CONSTRAINT fk_appointments_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE appointments
		   ADD CONSTRAINT fk_appointments_doctor
		   FOREIGN KEY (doctor_id) REFERENCES users(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE appointments
		   ADD CONSTRAINT fk_appointments_schedule
		   FOREIGN KEY (schedule_id) REFERENCES organisation_schedules(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
