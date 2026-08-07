package payments

import (
	"hospital-backend/pkg/db"
)

func Migrate(db db.Postgre) error {
	err := db.AutoMigrate(&Payments{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&PaymentAttempts{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&Refunds{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&WebhookEvents{})
	if err != nil {
		return err
	}
	return nil
}
