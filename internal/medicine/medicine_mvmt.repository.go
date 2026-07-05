package medicine

import (
	"hospital-backend/pkg/types"

	"gorm.io/gorm"
)

type RMedicineMvmt interface {
	CreateMedicineMvmtInBatch(db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error
}

func (r *MedicineRepo) CreateMedicineMvmtInBatch(db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error {
	return db.CreateInBatches(&medicineMvmt, len(medicineMvmt)).Error
}
