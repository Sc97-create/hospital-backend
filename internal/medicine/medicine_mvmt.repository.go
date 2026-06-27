package medicine

import "gorm.io/gorm"

type RMedicineMvmt interface {
	CreateMedicineMvmtInBatch(db *gorm.DB, medicineMvmt []MedicineStockMovements) error
}

func (r *MedicineRepo) CreateMedicineMvmtInBatch(db *gorm.DB, medicineMvmt []MedicineStockMovements) error {
	return db.CreateInBatches(&medicineMvmt, len(medicineMvmt)).Error
}
