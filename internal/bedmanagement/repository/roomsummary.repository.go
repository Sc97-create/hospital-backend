package repository

import (
	"hospital-backend/internal/bedmanagement/models"

	"gorm.io/gorm"
)

type RoomSummaryRepository interface {
	Create(roomSummary *models.RoomSummary) error
	Update(roomSummary *models.RoomSummary) error
	GetRoomSummaryByRoomType(roomTypeID string) (*models.RoomSummary, error)
}

type RoomSummaryDb struct {
	db *gorm.DB
}

func NewRoomSummaryDB(db *gorm.DB) *RoomSummaryDb {
	return &RoomSummaryDb{db: db}
}

func (r *RoomSummaryDb) Create(roomSummary *models.RoomSummary) error {
	err := r.db.Create(roomSummary).Error
	if err != nil {
		return err
	}
	return nil
}
func (r *RoomSummaryDb) Update(roomSummary *models.RoomSummary) error {
	err := r.db.Where("room_type_id = ?", roomSummary.RoomTypeID).Updates(roomSummary).Error
	if err != nil {
		return err
	}
	return nil
}
func (r *RoomSummaryDb) GetRoomSummaryByRoomType(roomTypeID string) (*models.RoomSummary, error) {
	var roomSummary models.RoomSummary
	err := r.db.Model(&models.RoomSummary{}).Select("id,total_rooms,total_floors,total_beds").Where("room_type_id = ?", roomTypeID).First(&roomSummary).Error
	if err != nil {
		return nil, err
	}
	return &roomSummary, nil
}
