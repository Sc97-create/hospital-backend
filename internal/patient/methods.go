package patient

import (
	"gorm.io/gorm"
)

type PatientRepo struct {
	db *gorm.DB
}

func NewPatientRepo(db *gorm.DB) *PatientRepo {
	return &PatientRepo{db: db}
}
