package medicine

import (
	"hospital-backend/pkg/types"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RMedicineMvmt interface {
	CreateMedicineMvmtInBatch(log *zap.Logger, db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error
}

func (r *MedicineRepo) CreateMedicineMvmtInBatch(log *zap.Logger, db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error {
	log = ensureLog(log)
	if len(medicineMvmt) == 0 {
		return nil
	}
	err := db.CreateInBatches(&medicineMvmt, len(medicineMvmt)).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CreateMedicineMvmtInBatch"), zap.Error(err))
		return err
	}
	return nil
}
