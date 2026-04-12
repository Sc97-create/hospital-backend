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
	CreateBatch(tx *gorm.DB, room *[]models.Room) error
	GetRoomByID(roomID string) (*models.Room, error)
}

func (b *RoomDB) CreateBatch(tx *gorm.DB, room *[]models.Room) error {
	err := tx.CreateInBatches(&room, 2).Error
	if err != nil {
		return err
	}
	return nil
}
func (b *RoomDB) GetRoomByID(roomID string) (room *models.Room, err error) {
	err = b.DB.Where("id = ?", roomID).First(&room).Error
	if err != nil {
		return nil, err
	}
	return room, nil
}
