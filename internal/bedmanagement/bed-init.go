package bedmanagement

import (
	bedRepo "hospital-backend/internal/bedmanagement/repository"
	bedService "hospital-backend/internal/bedmanagement/services"

	"gorm.io/gorm"
)

type BedContainer struct {
	RoomTypeService     *bedService.RoomTypeService
	RoomServices        *bedService.RoomService
	BedServices         *bedService.BedService
	RoomSummaryService  *bedService.RoomSummaryService
	BedAllotmentService *bedService.BedAllotmentService
}

func NewBedContainer(db *gorm.DB) *BedContainer {
	return &BedContainer{
		RoomTypeService:     bedService.NewRoomTypeService(bedRepo.NewRoomTypeDB(db)),
		RoomServices:        bedService.NewRoomService(db, bedRepo.NewRoomDB(db), bedService.NewRoomSummaryService(bedRepo.NewRoomSummaryDB(db))),
		BedServices:         bedService.NewBedService(db, bedRepo.NewBedDB(db), bedService.NewRoomSummaryService(bedRepo.NewRoomSummaryDB(db)), bedService.NewRoomTypeService(bedRepo.NewRoomTypeDB(db))),
		RoomSummaryService:  bedService.NewRoomSummaryService(bedRepo.NewRoomSummaryDB(db)),
		BedAllotmentService: bedService.NewBedAllotmentService(db, bedRepo.NewBedAllotmentDB(db)),
	}
}
