package appointments

import "hospital-backend/pkg/db"

func AutoMigrate(db db.Postgre) error {
	err := db.AutoMigrate(&Appointment{})
	if err != nil {
		return err
	}
	return nil
}
