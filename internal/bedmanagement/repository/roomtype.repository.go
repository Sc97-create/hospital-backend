package repository

import (
	"errors"
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
	CheckRoomType(roomType *models.RoomType) error
	GetRoomTypeData(roomTypeId string) (roomTypeResponse models.RoomType, err error)
	FindAllRoomTypes(organisationID string) ([]models.RoomType, error)
}

func (b *RoomTypeDB) Create(roomType *models.RoomType) error {
	err := b.DB.Create(&roomType).Error
	if err != nil {
		return err
	}
	return nil
}
func (b *RoomTypeDB) CheckRoomType(roomType *models.RoomType) error {
	count := int64(0)
	err := b.DB.Model(&models.RoomType{}).Where("name = ? AND organisation_id = ?", roomType.Name, roomType.OrganisationID).Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if count > 0 {
		return errors.New("ward already exits")
	}
	return nil
}
func (b *RoomTypeDB) GetRoomTypeData(roomTypeId string) (roomTypeResponse models.RoomType, err error) {
	err = b.DB.Model(&models.RoomType{}).Select("id,name,base_price").Where("id = ?", roomTypeId).First(&roomTypeResponse).Error
	if err != nil {
		return models.RoomType{}, err
	}
	return roomTypeResponse, nil
}
func (b *RoomTypeDB) FindAllRoomTypes(organisationID string) ([]models.RoomType, error) {
	var roomTypes []models.RoomType
	query := `select id,name,base_price from room_types where organisation_id = $1`
	err := b.DB.Model(&models.RoomType{}).Raw(query, organisationID).Scan(&roomTypes).Error
	if err != nil {
		return nil, err
	}
	return roomTypes, nil
}
