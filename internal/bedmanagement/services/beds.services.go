package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
	"hospital-backend/internal/bedmanagement/utils"
)

type BedService struct {
	BedRepo *repository.BedDb
}

func NewBedService(bedRepo *repository.BedDb) *BedService {
	return &BedService{BedRepo: bedRepo}
}

func (b *BedService) CreateBedSrv(bed dto.BedInfo) error {
	bedArray := []models.Bed{}
	for _, each := range bed.RoomNumber {
		beds := utils.GenerateBeds(bed.BedsPerRoom)
		bedModel := models.Bed{
			Beds:           beds,
			OrganisationID: bed.OrganisationID,
			RoomID:         each,
		}
		bedArray = append(bedArray, bedModel)
	}
	err := b.BedRepo.CreateBatch(&bedArray)
	if err != nil {
		return err
	}
	return nil
}
