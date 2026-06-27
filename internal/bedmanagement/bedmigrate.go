package bedmanagement

import (
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/pkg/db"
)

func Migrate(db db.Postgre) error {
	err := db.AutoMigrate(&models.RoomType{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&models.Room{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&models.Bed{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&models.RoomSummary{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&models.BedAllotment{})
	if err != nil {
		return err
	}
	return nil
}
