package medicine

import (
	"hospital-backend/internal/medicine/dto"
	"time"

	"github.com/google/uuid"
)

type MedicineService struct {
	Mrepo MedicineRepository
}

func NewMedicineService(Repo MedicineRepo) *MedicineService {
	return &MedicineService{Mrepo: &Repo}
}

func (MService *MedicineService) CreateMedicalSrv(payload dto.RequestPayload) error {
	medicineModel := new(Medicine)
	medicineModel.ID = uuid.New().String()
	medicineModel.BatchNumber = payload.BatchNumber
	medicineModel.ExpiryDate = payload.ExpiryDate
	medicineModel.Form = payload.Form
	medicineModel.Name = payload.Name
	medicineModel.Strength = payload.Strength
	medicineModel.CreatedAt = time.Now()
	medicineModel.CreatedBy = payload.UserID
	medicineModel.Quantity = payload.Quantity
	err := MService.Mrepo.Create(medicineModel)
	if err != nil {
		return err
	}
	return nil
}
func (Mservice *MedicineService) GetOne(id string) (*Medicine, error) {
	med, err := Mservice.Mrepo.FindOne(id)
	if err != nil {
		return nil, err
	}
	return med, nil
}
func (Mservice *MedicineService) GetMany(limit int, pageno int) (Med []Medicine, err error) {
	skip := 0
	if pageno != 0 {
		skip = (pageno - 1) * limit
	}
	Med, err = Mservice.GetMany(limit, skip)
	if err != nil {
		return
	}
	return
}
