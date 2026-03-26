package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
)

type RoomTypeService struct {
	RoomTypeRepo repository.RoomTypeRepository
}

func NewRoomTypeService(roomTypeRepo repository.RoomTypeRepository) *RoomTypeService {
	return &RoomTypeService{RoomTypeRepo: roomTypeRepo}
}
func (s *RoomTypeService) CreateRoomTypeSrv(roomType dto.RoomTypeInfo) (err error) {
	roomTypeDB := models.RoomType{
		Name:           roomType.Name,
		OrganisationID: roomType.OrganisationID,
		IsDefault:      roomType.IsDefault,
		BasePrice:      roomType.BasePrice,
	}
	err = s.RoomTypeRepo.Create(&roomTypeDB)
	if err != nil {
		return err
	}
	return nil
}
