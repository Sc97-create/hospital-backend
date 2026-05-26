package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
	"time"

	"github.com/google/uuid"
)

type RoomSummaryService struct {
	RoomSummaryRepo repository.RoomSummaryRepository
}

func NewRoomSummaryService(roomSummaryRepo repository.RoomSummaryRepository) *RoomSummaryService {
	return &RoomSummaryService{RoomSummaryRepo: roomSummaryRepo}
}

func (r *RoomSummaryService) CreateRoomSummary(roomSummary *models.RoomSummary) error {
	err := r.RoomSummaryRepo.Create(roomSummary)
	if err != nil {
		return err
	}
	return nil
}
func (r *RoomSummaryService) UpdateRoomSummary(roomSummary *models.RoomSummary) error {
	err := r.RoomSummaryRepo.Update(roomSummary)
	if err != nil {
		return err
	}
	return nil
}
func (r *RoomSummaryService) ToCreateRoomSummary(roomTypeID string, Rooms int, Floors int, organisationID string) *models.RoomSummary {
	return &models.RoomSummary{
		RoomTypeID:     roomTypeID,
		TotalRooms:     Rooms,
		TotalFloors:    Floors,
		OrganisationID: organisationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ID:             uuid.New().String(),
	}
}

func (r *RoomSummaryService) ToUpdateRoomSummary(params models.UpdateRoomSummaryParams) *models.RoomSummary {
	roomSummaryModel := models.RoomSummary{
		RoomTypeID: params.RoomTypeID,
	}
	if params.TotalRooms != 0 {
		roomSummaryModel.TotalRooms = params.TotalRooms
	}
	if params.TotalFloors != 0 {
		roomSummaryModel.TotalFloors = params.TotalFloors
	}
	if params.TotalBeds != 0 {
		roomSummaryModel.TotalBeds = params.TotalBeds
	}
	return &roomSummaryModel
}

func (r *RoomSummaryService) GetRoomSummaryByRoomType(roomTypeID string) (dto.RoomSummaryResponse, error) {

	roomsummary, err := r.RoomSummaryRepo.GetRoomSummaryByRoomType(roomTypeID)
	if err != nil {
		return dto.RoomSummaryResponse{}, err
	}
	return r.ToRoomSummaryResponse(roomsummary), nil
}
func (r *RoomSummaryService) ToRoomSummaryResponse(roomSummary *models.RoomSummary) dto.RoomSummaryResponse {
	return dto.RoomSummaryResponse{
		TotalBeds:   roomSummary.TotalBeds,
		TotalFloors: roomSummary.TotalFloors,
		TotalRooms:  roomSummary.TotalRooms,
	}
}
