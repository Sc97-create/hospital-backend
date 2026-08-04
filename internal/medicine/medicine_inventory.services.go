package medicine

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SMedicineInventory struct {
	MedInventory RMedicineInventory
}

func NewSMedicineInventory(medInventory RMedicineInventory) *SMedicineInventory {
	return &SMedicineInventory{MedInventory: medInventory}
}

func (s *SMedicineInventory) CreateMedicineInventory(log *zap.Logger, db *gorm.DB, medicineInventory []MedicineInventory) error {
	return s.MedInventory.CreateInventoryInBatch(ensureLog(log), db, medicineInventory)
}

func (s *SMedicineInventory) UpdateMedInventoryStock(log *zap.Logger, tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error {
	return s.MedInventory.UpdateMedInventoryStock(ensureLog(log), tx, medicineInventoryID, dispensedQty)
}
