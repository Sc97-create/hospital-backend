package services

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/models"
	"hospital-backend/internal/bedmanagement/repository"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BedAllotmentService struct {
	DB               *gorm.DB
	BedAllotmentRepo repository.BedAllotmentRepository
}

func NewBedAllotmentService(db *gorm.DB, bedAllotmentRepo repository.BedAllotmentRepository) *BedAllotmentService {
	return &BedAllotmentService{DB: db, BedAllotmentRepo: bedAllotmentRepo}
}

func (s *BedAllotmentService) CreateBedAllotment(bedAllotment dto.BedAllotmentCreatePayload) error {
	bedAllotmentModel := s.ToBedAllotmentModel(bedAllotment)
	err := s.BedAllotmentRepo.Create(&bedAllotmentModel)
	if err != nil {
		return err
	}
	return nil
}
func (s *BedAllotmentService) ToBedAllotmentModel(bedAllotment dto.BedAllotmentCreatePayload) models.BedAllotment {
	return models.BedAllotment{
		ID:        uuid.New().String(),
		PatientID: bedAllotment.PatientID,
		BedID:     bedAllotment.BedID,
		RoomID:    bedAllotment.RoomID,
		CreatedAt: time.Now(),
		//OrganisationID: bedAllotment.OrganisationID,
		RoomTypeID:   bedAllotment.RoomTypeID,
		Charges:      bedAllotment.BedCharges,
		DischargedAt: bedAllotment.DischargeAt,
	}
}
