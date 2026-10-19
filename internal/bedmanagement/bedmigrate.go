package bedmanagement

import (
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

func Migrate(database db.Postgre) error {
	if err := database.AutoMigrate(&models.RoomType{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&models.Room{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&models.Bed{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&models.RoomSummary{}); err != nil {
		return err
	}
	if err := database.AutoMigrate(&models.BedAllotment{}); err != nil {
		return err
	}
	return ensureBedForeignKeys(database.GormDriver)
}

func ensureBedForeignKeys(gdb *gorm.DB) error {
	return db.EnsureForeignKeys(gdb, []string{
		`ALTER TABLE room_types
		   ADD CONSTRAINT fk_room_types_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE rooms
		   ADD CONSTRAINT fk_rooms_room_type
		   FOREIGN KEY (room_type_id) REFERENCES room_types(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE rooms
		   ADD CONSTRAINT fk_rooms_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE beds
		   ADD CONSTRAINT fk_beds_room
		   FOREIGN KEY (room_id) REFERENCES rooms(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE beds
		   ADD CONSTRAINT fk_beds_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE bed_allotments
		   ADD CONSTRAINT fk_bed_allotments_bed
		   FOREIGN KEY (bed_id) REFERENCES beds(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE bed_allotments
		   ADD CONSTRAINT fk_bed_allotments_room
		   FOREIGN KEY (room_id) REFERENCES rooms(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE bed_allotments
		   ADD CONSTRAINT fk_bed_allotments_room_type
		   FOREIGN KEY (room_type_id) REFERENCES room_types(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE bed_allotments
		   ADD CONSTRAINT fk_bed_allotments_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
