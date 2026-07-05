package medicine

import (
	"hospital-backend/pkg/types"

	"gorm.io/gorm"
)

type SMedicineMvmt struct {
	MedMvmt RMedicineMvmt
}

func NewMedicineMvmt(medMvmt RMedicineMvmt) *SMedicineMvmt {
	return &SMedicineMvmt{MedMvmt: medMvmt}
}
func (s *SMedicineMvmt) CreateMedicineMvmt(db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error {
	return s.MedMvmt.CreateMedicineMvmtInBatch(db, medicineMvmt)
}
