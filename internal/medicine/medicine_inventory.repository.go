package medicine

import "gorm.io/gorm"

type RMedicineInventory interface {
	CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error
}

func (r *MedicineRepo) CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error {
	return db.CreateInBatches(&medicineInventory, len(medicineInventory)).Error
}
