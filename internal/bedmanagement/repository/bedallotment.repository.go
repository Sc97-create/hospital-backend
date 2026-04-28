package repository

import (
	"hospital-backend/internal/bedmanagement/models"

	"gorm.io/gorm"
)

type BedAllotmentDB struct {
	DB *gorm.DB
}

func NewBedAllotmentDB(db *gorm.DB) *BedAllotmentDB {
	return &BedAllotmentDB{DB: db}
}

type BedAllotmentRepository interface {
	Create(bedAllotment *models.BedAllotment) error
}

func (b *BedAllotmentDB) Create(bedAllotment *models.BedAllotment) error {
	err := b.DB.Create(bedAllotment).Error
	if err != nil {
		return err
	}
	return nil
}
