package repository

import (
	"hospital-backend/internal/bedmanagement/models"

	"gorm.io/gorm"
)

type BedDb struct {
	db *gorm.DB
}

func NewBedDB(db *gorm.DB) *BedDb {
	return &BedDb{db: db}
}

type BedRepository interface {
	Create(bed *models.Bed) error
}

func (b *BedDb) CreateBatch(bed *[]models.Bed) error {
	err := b.db.CreateInBatches(&bed, 2).Error
	if err != nil {
		return err
	}
	return nil
}
