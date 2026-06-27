package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
	"hospital-backend/internal/bedmanagement/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BedService struct {
	DB                 *gorm.DB
	BedRepo            *repository.BedDb
	RoomSummaryService *RoomSummaryService
	RoomTypeService    *RoomTypeService
}

func NewBedService(db *gorm.DB, bedRepo *repository.BedDb, roomSummaryService *RoomSummaryService, roomTypeService *RoomTypeService) *BedService {
	return &BedService{DB: db, BedRepo: bedRepo, RoomSummaryService: roomSummaryService, RoomTypeService: roomTypeService}
}

func (b *BedService) CreateBedSrv(bed dto.CreateBed) error {
	bedArray := b.CreateBedModel(bed)
	//create room summary model
	b.DB.Transaction(func(tx *gorm.DB) error {
		err := b.BedRepo.CreateBatch(tx, &bedArray)
		if err != nil {
			return err
		}
		roomSummaryParams := models.UpdateRoomSummaryParams{}
		roomSummaryParams.TotalBeds = len(bedArray)
		roomSummaryParams.RoomTypeID = bed.RoomTypeID
		roomSummaryModel := b.RoomSummaryService.ToUpdateRoomSummary(roomSummaryParams)
		err = b.RoomSummaryService.UpdateRoomSummary(roomSummaryModel)
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}
func (b *BedService) GenerateBeds(bed dto.BedGenerate) (map[string][]dto.BedResponse, dto.RoomSummaryResponse, error) {
	roomType, err := b.RoomTypeService.GetRoomTypeData(bed.RoomTypeID)
	if err != nil {
		return nil, dto.RoomSummaryResponse{}, err
	}
	bedMap, totalbeds, err := b.ToBedModel(bed, roomType)
	if err != nil {
		return nil, dto.RoomSummaryResponse{}, err
	}
	roomsummary, err := b.RoomSummaryService.GetRoomSummaryByRoomType(bed.RoomTypeID)
	if err != nil {
		return nil, dto.RoomSummaryResponse{}, err
	}
	if roomsummary.TotalBeds == 0 {
		roomsummary.TotalBeds = totalbeds
	}
	return bedMap, roomsummary, nil
}
func (b *BedService) CreateBedModel(bed dto.CreateBed) []models.Bed {
	bedArray := []models.Bed{}

	for _, each := range bed.Beds {
		bedModel := models.Bed{}
		bedModel.Status = models.StatusAvailable
		bedModel.RoomID = each.RoomID
		bedModel.OrganisationID = bed.OrganisationID
		for _, eachbed := range each.BedsArray {
			bedModel.ID = uuid.New().String()
			bedModel.Beds = eachbed
			bedArray = append(bedArray, bedModel)

		}

	}
	return bedArray
}
func (b *BedService) ToBedModel(bed dto.BedGenerate, roomType dto.RoomTypeResponse) (map[string][]dto.BedResponse, int, error) {
	bedMap := make(map[string][]dto.BedResponse)
	count := 0

	for _, each := range bed.RoomNumber {
		bedArray := []dto.BedResponse{}
		beds := utils.GenerateBeds(bed.BedsPerRoom, roomType.RoomTypeName)
		count += len(beds)
		bedModel := dto.BedResponse{
			BedNumber:  beds,
			RoomNumber: each,
		}
		bedArray = append(bedArray, bedModel)

		bedMap[each] = bedArray

	}
	return bedMap, count, nil
}
func (s *BedService) FindAllAvailableBeds(organisationID string, limit string, offset string, roomID string) ([]models.Bed, error) {
	skip, limitInt := utils.CalculateSkip(offset, limit)
	return s.BedRepo.FindAllAvailableBeds(organisationID, limitInt, skip, roomID)
}
