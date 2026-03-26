package bedmanagement

import (
	bedRepo "hospital-backend/internal/bedmanagement/repository"
	bedService "hospital-backend/internal/bedmanagement/services"

	"gorm.io/gorm"
)

type BedContainer struct {
	RoomTypeService *bedService.RoomTypeService
	RoomServices    *bedService.RoomService
	BedServices     *bedService.BedService
}

func NewBedContainer(db *gorm.DB) *BedContainer {
	return &BedContainer{
		RoomTypeService: bedService.NewRoomTypeService(bedRepo.NewRoomTypeDB(db)),
		RoomServices:    bedService.NewRoomService(bedRepo.NewRoomDB(db)),
		BedServices:     bedService.NewBedService(bedRepo.NewBedDB(db)),
	}
}
