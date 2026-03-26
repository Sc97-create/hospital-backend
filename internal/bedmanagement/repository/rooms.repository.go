package repository

import (
	"hospital-backend/internal/bedmanagement/models"

	"gorm.io/gorm"
)

type RoomDB struct {
	DB *gorm.DB
}

func NewRoomDB(db *gorm.DB) *RoomDB {
	return &RoomDB{DB: db}
}

type RoomRepository interface {
	CreateBatch(room *[]models.Room) error
}

func (b *RoomDB) CreateBatch(room *[]models.Room) error {
	err := b.DB.CreateInBatches(&room, 2).Error
	if err != nil {
		return err
	}
	return nil
}
