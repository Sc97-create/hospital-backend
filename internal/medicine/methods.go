package medicine

import (
	"gorm.io/gorm"
)

type MedicineRepo struct {
	db *gorm.DB
}

func NewMedicineRepo(db *gorm.DB) *MedicineRepo {
	return &MedicineRepo{db: db}
}
