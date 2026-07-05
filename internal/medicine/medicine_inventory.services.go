package medicine

import "gorm.io/gorm"

type SMedicineInventory struct {
	MedInventory RMedicineInventory
}

func NewSMedicineInventory(medInventory RMedicineInventory) *SMedicineInventory {
	return &SMedicineInventory{MedInventory: medInventory}
}
func (s *SMedicineInventory) CreateMedicineInventory(db *gorm.DB, medicineInventory []MedicineInventory) error {
	return s.MedInventory.CreateInventoryInBatch(db, medicineInventory)
}
