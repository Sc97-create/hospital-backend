package patient

import (
	"errors"
	"hospital-backend/internal/patient/dto"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type PatientService struct {
	PRepo PatientRepository
}

func NewPatientService(p PatientRepository) *PatientService {
	return &PatientService{PRepo: p}
}
func (p *PatientService) CreatePatientSrv(payload dto.PatientInfo) (string, error) {
	if payload.FirstName == "" {
		err := errors.New("please provide name")
		return "", err
	}
	if payload.Gender == "" {
		err := errors.New("please provide valid gender")
		return "", err
	}
	age, _ := strconv.Atoi(payload.Age)
	if age < 0 {
		err := errors.New("age should be greater then 0")
		return "", err
	}
	weight, _ := strconv.ParseFloat(payload.Weight, 64)
	if weight <= 0.0 {
		err := errors.New("weight should not be 0")
		return "", err
	}
	patientModel := Patient{
		ID:              uuid.New().String(),
		Name:            payload.FirstName,
		Age:             age,
		Weight:          int(weight),
		Symptoms:        payload.Symptoms,
		EmailID:         payload.EmailID,
		CreatedBy:       payload.UserID,
		AdmissionDate:   time.Now(),
		ActiveCondition: payload.ActiveCondition,
		Gender:          payload.Gender,
		MobileNumber:    payload.MobileNumber,
		OrganisationID:  payload.OrganisationID,
		DoctorID:        payload.DoctorID,
	}
	err := p.PRepo.Create(&patientModel)
	if err != nil {
		return "", err
	}
	return patientModel.ID, nil
}
func (p *PatientService) FindMany(limit int, pageno int) (patient []Patient, err error) {
	skip := 0
	if pageno != 0 {
		skip = (pageno - 1) * limit
	}
	patient, err = p.PRepo.ReadMany(limit, skip)
	if err != nil {
		return
	}
	return
}
func (p *PatientService) FindOne(id string) (pat Patient, err error) {
	pat, err = p.PRepo.ReadOne(id)
	if err != nil {
		return
	}
	return
}
