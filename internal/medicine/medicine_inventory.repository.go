package medicine

import (
	"fmt"

	"gorm.io/gorm"
)

type RMedicineInventory interface {
	CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error
	GetMedByBatchNo(query string, args ...any) (MedicineInventory, error)
	// UpdateMedInventoryStock atomically decrements current_stock_units by dispensedQty.
	// Returns error if stock is insufficient (current_stock_units < dispensedQty).
	UpdateMedInventoryStock(tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error
}

func (r *MedicineRepo) CreateInventoryInBatch(db *gorm.DB, medicineInventory []MedicineInventory) error {
	return db.CreateInBatches(&medicineInventory, len(medicineInventory)).Error
}

func (r *MedicineRepo) GetMedByBatchNo(query string, args ...any) (MedicineInventory, error) {
	var medInventory MedicineInventory
	err := r.db.Raw(query, args...).Scan(&medInventory).Error
	if err != nil {
		return MedicineInventory{}, err
	}
	return medInventory, nil
}

// UpdateMedInventoryStock decrements stock atomically — safe under concurrent webhook calls.
// The WHERE current_stock_units >= dispensedQty prevents negative stock.
func (r *MedicineRepo) UpdateMedInventoryStock(tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error {
	result := tx.Model(&MedicineInventory{}).
		Where("id = ? AND current_stock_units >= ?", medicineInventoryID, dispensedQty).
		Update("current_stock_units", gorm.Expr("current_stock_units - ?", dispensedQty))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient stock for medicine inventory %s", medicineInventoryID)
	}
	return nil
}
