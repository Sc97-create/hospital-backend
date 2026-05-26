package medmigration

import (
	"hospital-backend/internal/medicine"
	"hospital-backend/pkg/db"
)

func Automigrate(db db.Postgre) error {
	err := db.AutoMigrate(&medicine.Medicine{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&medicine.MedicineInventory{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&medicine.MedicineStockMovements{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&medicine.MPurchaseEntry{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&medicine.Supplier{})
	if err != nil {
		return err
	}
	return nil

}
