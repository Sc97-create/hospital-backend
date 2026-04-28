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
	CreateBatch(db *gorm.DB, bed *models.Bed) error
	FindAllAvailableBeds(organisationID string, limit int, offset int) ([]models.Bed, error)
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
func (b *BedDb) FindAllAvailableBeds(organisationID string, limit int, offset int, roomID string) ([]models.Bed, error) {
	var beds []models.Bed
	query := `select id,beds,status from beds where status=$1 and organisation_id=$2 and room_id=$3 limit $4 offset $5`
	err := b.db.Model(models.Bed{}).Raw(query, models.StatusAvailable, organisationID, roomID, limit, offset).Scan(&beds).Error
	if err != nil {
		return nil, err
	}
	return beds, nil
}
