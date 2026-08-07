package medicine

import (
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RMedicineInventory interface {
	CreateInventoryInBatch(log *zap.Logger, db *gorm.DB, medicineInventory []MedicineInventory) error
	GetMedByBatchNo(log *zap.Logger, query string, args ...any) (MedicineInventory, error)
	// UpdateMedInventoryStock atomically decrements current_stock_units by dispensedQty.
	// Returns ErrInsufficientStock if stock is insufficient (current_stock_units < dispensedQty).
	UpdateMedInventoryStock(log *zap.Logger, tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error
}

func (r *MedicineRepo) CreateInventoryInBatch(log *zap.Logger, db *gorm.DB, medicineInventory []MedicineInventory) error {
	log = ensureLog(log)
	if len(medicineInventory) == 0 {
		return nil
	}
	err := db.CreateInBatches(&medicineInventory, len(medicineInventory)).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CreateInventoryInBatch"), zap.Error(err))
		return err
	}
	return nil
}

func (r *MedicineRepo) GetMedByBatchNo(log *zap.Logger, query string, args ...any) (MedicineInventory, error) {
	log = ensureLog(log)
	var medInventory MedicineInventory
	err := r.db.Raw(query, args...).Scan(&medInventory).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "GetMedByBatchNo"), zap.Error(err))
		return MedicineInventory{}, err
	}
	return medInventory, nil
}

func (r *MedicineRepo) UpdateMedInventoryStock(log *zap.Logger, tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error {
	log = ensureLog(log)
	result := tx.Model(&MedicineInventory{}).
		Where("id = ? AND current_stock_units >= ?", medicineInventoryID, dispensedQty).
		Update("current_stock_units", gorm.Expr("current_stock_units - ?", dispensedQty))
	if result.Error != nil {
		log.Error("medicine repo error",
			zap.String("op", "UpdateMedInventoryStock"),
			zap.String("medicine_inventory_id", medicineInventoryID),
			zap.Error(result.Error),
		)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Error("medicine stock decrement failed",
			zap.String("medicine_inventory_id", medicineInventoryID),
			zap.Int64("dispensed_qty", dispensedQty),
			zap.String("reason", "insufficient_stock"),
		)
		return wrapError.ErrInsufficientStock
	}
	return nil
}
