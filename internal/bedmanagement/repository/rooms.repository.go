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
	FindAllAvailableRooms(organisationID string, limit int, offset int) ([]models.Room, error)
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
func (b *RoomDB) FindAllAvailableRooms(organistionID string, limit int, offset int) ([]models.Room, error) {
	var rooms []models.Room
	query := `select id,room_number,status,floors from rooms where status=$1 and organisation_id=$2 limit $3 offset $4`
	err := b.DB.Model(models.Room{}).Raw(query, models.StatusAvailable, organistionID, limit, offset).Scan(&rooms).Error
	if err != nil {
		return nil, err
	}
	return rooms, nil
}
