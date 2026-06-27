package medicine

import "gorm.io/gorm"

type RMedicineInventory interface {
	CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error
	GetMedicinesByMedID(medicineID string) (medInventory []MixedMedInventory, err error)
}

func (r *MedicineRepo) CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error {
	return db.CreateInBatches(&medicineInventory, len(medicineInventory)).Error
}

func (r *MedicineRepo) GetMedicinesByMedID(medicineID string) (medInventory []MixedMedInventory, err error) {

	return
}
