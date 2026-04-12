package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
	"hospital-backend/internal/bedmanagement/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomService struct {
	DB                 *gorm.DB
	RoomRepo           repository.RoomRepository
	RoomSummaryService *RoomSummaryService
}

func NewRoomService(db *gorm.DB, roomRepo repository.RoomRepository, roomSummaryService *RoomSummaryService) *RoomService {
	return &RoomService{DB: db, RoomRepo: roomRepo, RoomSummaryService: roomSummaryService}
}
func (i *RoomService) CreateBatchRooms(payloadReq dto.RoomRequest) ([]models.Room, error) {
	totalRooms := payloadReq.Floor * (payloadReq.RoomPerFloor - payloadReq.StartingPerFloor + 1)
	roomNumbers := utils.GenerateRoomNumber(payloadReq.Prefix, int(payloadReq.Floor), int(payloadReq.RoomPerFloor), int(payloadReq.StartingPerFloor))
	roomModel := i.ToRoomModel(roomNumbers, payloadReq.RoomTypeID, payloadReq.OrganisationID)
	err := i.DB.Transaction(func(tx *gorm.DB) error {
		err := i.RoomRepo.CreateBatch(tx, &roomModel)
		if err != nil {
			return err
		}
		RoomSummaryModel := i.RoomSummaryService.ToCreateRoomSummary(payloadReq.RoomTypeID, int(totalRooms), int(payloadReq.Floor), payloadReq.OrganisationID)
		err = i.RoomSummaryService.CreateRoomSummary(RoomSummaryModel)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return roomModel, nil
}
func (i *RoomService) ToRoomModel(roomNumbers map[int][]string, roomTypeID string, organisationID string) []models.Room {
	roomModel := []models.Room{}
	for key, each := range roomNumbers {
		for _, rooms := range each {
			roomModel = append(roomModel, models.Room{
				RoomNumber:     rooms,
				RoomTypeID:     roomTypeID,
				OrganisationID: organisationID,
				Floors:         key,
				Status:         models.StatusAvailable,
				ID:             uuid.New().String(),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
		}
	}
	return roomModel
}
