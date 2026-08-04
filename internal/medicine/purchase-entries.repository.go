package medicine

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RPurchaseEntry interface {
	CreatePurchaseEntry(log *zap.Logger, db *gorm.DB, purchaseEntry *MPurchaseEntry) error
}

func (r *MedicineRepo) CreatePurchaseEntry(log *zap.Logger, db *gorm.DB, purchaseEntry *MPurchaseEntry) error {
	log = ensureLog(log)
	err := db.Create(&purchaseEntry).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CreatePurchaseEntry"), zap.Error(err))
		return err
	}
	return nil
}
