package medicine

import "gorm.io/gorm"

type RMedicineInventory interface {
	CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error
	GetMedicinesByMedID(medicineID string) (medInventory []MixedMedInventory, err error)
	GetMedByBatchNo(query string, args ...any) (MedicineInventory, error)
}

func (r *MedicineRepo) CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error {
	return db.CreateInBatches(&medicineInventory, len(medicineInventory)).Error
}

func (r *MedicineRepo) GetMedicinesByMedID(medicineID string) (medInventory []MixedMedInventory, err error) {

	return
}
func (r *MedicineRepo) GetMedByBatchNo(query string, args ...any) (MedicineInventory, error) {
	var medInventory MedicineInventory
	err := r.db.Raw(query, args...).Scan(&medInventory).Error
	if err != nil {
		return MedicineInventory{}, err
	}
	return medInventory, nil
} // can be used for future
