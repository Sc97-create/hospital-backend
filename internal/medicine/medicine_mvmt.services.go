package medicine

import "gorm.io/gorm"

type SMedicineMvmt struct {
	MedMvmt RMedicineMvmt
}

func NewMedicineMvmt(medMvmt RMedicineMvmt) *SMedicineMvmt {
	return &SMedicineMvmt{MedMvmt: medMvmt}
}
func (s *SMedicineMvmt) CreateMedicineMvmt(db *gorm.DB, medicineMvmt []MedicineStockMovements) error {
	return s.MedMvmt.CreateMedicineMvmtInBatch(db, medicineMvmt)
}
