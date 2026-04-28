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
	age, weight, err := p.ValidatePatient(payload)
	if err != nil {
		return "", err
	}
	patientModel, err := p.ToPatientModel(age, weight, payload)
	if err != nil {
		return "", err
	}
	err = p.PRepo.Create(&patientModel)
	if err != nil {
		return "", err
	}
	return patientModel.ID, nil
}
func (p *PatientService) ValidatePatient(payload dto.PatientInfo) (int, float64, error) {
	if payload.FirstName == "" {
		err := errors.New("please provide name")
		return 0, 0.0, err
	}
	if payload.Gender == "" {
		err := errors.New("please provide valid gender")
		return 0, 0.0, err
	}
	age, _ := strconv.Atoi(payload.Age)
	if age < 0 {
		err := errors.New("age should be greater then 0")
		return 0, 0.0, err
	}
	weight, _ := strconv.ParseFloat(payload.Weight, 64)
	if weight <= 0.0 {
		err := errors.New("weight should not be 0")
		return 0, 0.0, err
	}
	return age, weight, nil
}
func (p *PatientService) FindMany(limit string, pageno string, organisationID string) (patientResp []dto.PatientResponse, err error) {
	limitInt, skip := p.GetPageSkip(limit, pageno)
	patient, err := p.PRepo.ReadMany(limitInt, skip, organisationID)
	if err != nil {
		return
	}
	patientResp = p.maptopatientResponse(patient)
	return
}
func (p *PatientService) ToPatientModel(age int, weight float64, payload dto.PatientInfo) (patientModel Patient, err error) {
	patientCode, err := p.GeneratePatientCode(payload.OrganisationID)
	if err != nil {
		return
	}
	patientModel = Patient{
		ID:              uuid.New().String(),
		PatientCode:     patientCode,
		FirstName:       payload.FirstName,
		LastName:        payload.LastName,
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
		Status:          StatusAdmitted,
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
func (p *PatientService) GeneratePatientCode(organisationID string) (string, error) {
	count, err := p.PRepo.Count(organisationID)
	if err != nil {
		return "", err
	}
	return "PAT" + strconv.FormatInt(count+1, 10), nil
}
func (p *PatientService) GetPageSkip(limit string, pageno string) (int, int) {
	skip := 0
	limitInt, _ := strconv.Atoi(limit)
	pagenoInt, _ := strconv.Atoi(pageno)
	if pagenoInt != 0 {
		skip = (pagenoInt - 1) * limitInt
	}
	return limitInt, skip
}
func (p *PatientService) maptopatientResponse(patient []Patient) []dto.PatientResponse {
	patientResponse := []dto.PatientResponse{}
	for _, each := range patient {
		patientResponse = append(patientResponse, dto.PatientResponse{
			PatientID:     each.ID,
			PatientCode:   each.PatientCode,
			PatientName:   each.FirstName,
			PatientWeight: each.Weight,
			PatientGender: each.Gender,
			PatientPhone:  each.MobileNumber,
			PatientEmail:  each.EmailID,
			PatientAge:    each.Age,
			PatientStatus: string(each.Status),
			AdmissionDate: each.AdmissionDate,
		})
	}
	return patientResponse
}
