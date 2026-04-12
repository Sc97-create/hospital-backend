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
	Create(db *gorm.DB, bed *models.Bed) error
}

func (b *BedDb) CreateBatch(tx *gorm.DB, bed *[]models.Bed) error {
	err := tx.CreateInBatches(&bed, 2).Error
	if err != nil {
		return err
	}
	return nil
}
func (b *BedDb) CheckIfExist(bed *models.Bed) error {

	return nil
}
