package medicine

import (
	"hospital-backend/pkg/types"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SMedicineMvmt struct {
	MedMvmt RMedicineMvmt
}

func NewMedicineMvmt(medMvmt RMedicineMvmt) *SMedicineMvmt {
	return &SMedicineMvmt{MedMvmt: medMvmt}
}

func (s *SMedicineMvmt) CreateMedicineMvmt(log *zap.Logger, db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error {
	return s.MedMvmt.CreateMedicineMvmtInBatch(ensureLog(log), db, medicineMvmt)
}
