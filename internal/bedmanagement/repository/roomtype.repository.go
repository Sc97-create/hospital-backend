package repository

import (
	"hospital-backend/internal/bedmanagement/models"

	"gorm.io/gorm"
)

type RoomTypeDB struct {
	DB *gorm.DB
}

func NewRoomTypeDB(db *gorm.DB) *RoomTypeDB {
	return &RoomTypeDB{DB: db}
}

type RoomTypeRepository interface {
	Create(roomType *models.RoomType) error
}

func (b *RoomTypeDB) Create(roomType *models.RoomType) error {
	err := b.DB.Create(&roomType).Error
	if err != nil {
		return err
	}
	return nil
}
