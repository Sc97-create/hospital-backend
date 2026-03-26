package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
	"hospital-backend/internal/bedmanagement/utils"
	"time"

	"github.com/google/uuid"
)

type RoomService struct {
	RoomRepo repository.RoomRepository
}

func NewRoomService(roomRepo repository.RoomRepository) *RoomService {
	return &RoomService{RoomRepo: roomRepo}
}
func (i *RoomService) CreateBatchRooms(payloadReq dto.RoomRequest) (err error) {
	totalRooms := payloadReq.Floor * (payloadReq.RoomPerFloor - payloadReq.StartingPerFloor + 1)
	roomModel := make([]models.Room, 0, totalRooms)
	roomNumbers := utils.GenerateRoomNumber(payloadReq.Prefix, payloadReq.Floor, payloadReq.RoomPerFloor, payloadReq.StartingPerFloor)
	
	for key, each := range roomNumbers {
		for _, rooms := range each {
			roomModel = append(roomModel, models.Room{
				RoomNumber:     rooms,
				RoomTypeID:     payloadReq.RoomTypeID,
				OrganisationID: payloadReq.OrganisationID,
				Floors:         key,
				Status:         "available",
				ID:             uuid.New().String(),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
		}
	}

	err = i.RoomRepo.CreateBatch(&roomModel)
	if err != nil {
		return err
	}

	return
}
