package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"

	"github.com/google/uuid"
)

type RoomTypeService struct {
	RoomTypeRepo repository.RoomTypeRepository
}

func NewRoomTypeService(roomTypeRepo repository.RoomTypeRepository) *RoomTypeService {
	return &RoomTypeService{RoomTypeRepo: roomTypeRepo}
}
func (s *RoomTypeService) CreateRoomTypeSrv(roomType dto.RoomTypeInfo) (roomTypeResponse dto.RoomTypeResponse, err error) {
	roomTypeDB := s.CreateRoomTypeModel(roomType)
	err = s.RoomTypeRepo.CheckRoomType(&roomTypeDB)
	if err != nil {
		return dto.RoomTypeResponse{}, err
	}
	err = s.RoomTypeRepo.Create(&roomTypeDB)
	if err != nil {
		return dto.RoomTypeResponse{}, err
	}
	roomTypeResponse.RoomTypeID = roomTypeDB.ID
	roomTypeResponse.RoomTypeName = roomType.Name
	return roomTypeResponse, nil
}
func (s *RoomTypeService) CreateRoomTypeModel(roomType dto.RoomTypeInfo) models.RoomType {
	roomTypeDB := models.RoomType{
		ID:             uuid.New().String(),
		Name:           roomType.Name,
		OrganisationID: roomType.OrganisationID,
		IsDefault:      roomType.IsDefault,
		BasePrice:      roomType.BasePrice,
	}
	return roomTypeDB
}
func (s *RoomTypeService) GetRoomTypeData(roomTypeId string) (roomTypeResponse dto.RoomTypeResponse, err error) {
	roomTypeDB, err := s.RoomTypeRepo.GetRoomTypeData(roomTypeId)
	if err != nil {
		return dto.RoomTypeResponse{}, err
	}
	roomTypeResponse.RoomTypeID = roomTypeDB.ID
	roomTypeResponse.RoomTypeName = roomTypeDB.Name
	roomTypeResponse.BasePrice = roomTypeDB.BasePrice
	return roomTypeResponse, nil
}
