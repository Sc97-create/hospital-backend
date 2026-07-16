package billing

import (
	"hospital-backend/pkg/db"
)

func Migrate(db db.Postgre) error {
	err := db.AutoMigrate(&Invoice{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&InvoiceItem{})
	if err != nil {
		return err
	}
	return nil
}
