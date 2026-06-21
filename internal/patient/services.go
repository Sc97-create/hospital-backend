package patient

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/patient/dto"
	"math/rand"
	"strconv"
	"strings"
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
	if payload.Name == "" {
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
func (p *PatientService) FindMany(limit string, pageno string, organisationID string) (patientResp []dto.PatientResponse, total int64, err error) {
	limitInt, skip := p.GetPageSkip(limit, pageno)
	patient, err := p.PRepo.ReadMany(limitInt, skip, organisationID)
	if err != nil {
		return
	}
	total, err = p.PRepo.Count(organisationID)
	if err != nil {
		return
	}
	patientResp = p.arraymaptopatientResponse(patient)
	return
}
func (p *PatientService) ToPatientModel(age int, weight float64, payload dto.PatientInfo) (patientModel Patient, err error) {
	patientModel = Patient{
		ID:             uuid.New().String(),
		UHID:           p.createPatientCode(),
		Name:           payload.Name,
		Age:            age,
		Weight:         int(weight),
		EmailID:        payload.EmailID,
		CreatedBy:      payload.UserID,
		LastVisitDate:  time.Now(),
		Gender:         payload.Gender,
		MobileNumber:   payload.MobileNumber,
		OrganisationID: payload.OrganisationID,
		Status:         StatusActive,
		Address:        payload.Address,
		BloodGroup:     payload.BloodGroup,
		CreatedAt:      time.Now(),
	}
	return
}
func (p *PatientService) createPatientCode() string {
	return fmt.Sprintf("%s-%d", Code, rand.Intn(1000))
}
func (p *PatientService) FindOne(id string) (pat dto.PatientResponse, err error) {
	patient, err := p.PRepo.ReadOne(id)
	if err != nil {
		return
	}

	pat = p.maptopatientResponse(patient)

	return
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
func (p *PatientService) arraymaptopatientResponse(patient []Patient) []dto.PatientResponse {
	patientResponse := []dto.PatientResponse{}
	for _, each := range patient {
		patientResponse = append(patientResponse, dto.PatientResponse{
			PatientID:      each.ID,
			PatientCode:    each.UHID,
			PatientName:    each.Name,
			PatientWeight:  each.Weight,
			PatientGender:  each.Gender,
			PatientPhone:   each.MobileNumber,
			PatientEmail:   each.EmailID,
			PatientAge:     each.Age,
			PatientStatus:  string(each.Status),
			PatientBG:      each.BloodGroup,
			PatientLVD:     each.LastVisitDate,
			PatientAddress: each.Address,
		})
	}
	return patientResponse
}
func (p *PatientService) maptopatientResponse(patient Patient) dto.PatientResponse {
	waitingTime := p.formatWaitingTime(patient.LastVisitDate)
	return dto.PatientResponse{
		PatientID:      patient.ID,
		PatientCode:    patient.UHID,
		PatientName:    patient.Name,
		PatientWeight:  patient.Weight,
		PatientGender:  patient.Gender,
		PatientPhone:   patient.MobileNumber,
		PatientEmail:   patient.EmailID,
		PatientAge:     patient.Age,
		PatientStatus:  string(patient.Status),
		PatientBG:      patient.BloodGroup,
		PatientLVD:     patient.LastVisitDate,
		PatientAddress: patient.Address,
		WaitingTime:    waitingTime,
	}
}
func (p *PatientService) formatWaitingTime(lastVisit time.Time) string {

	duration := time.Since(lastVisit)

	minutes := duration.Minutes()
	hours := duration.Hours()
	days := hours / 24

	// More than 30 days
	if days >= 30 {
		return "0"
	}

	// More than 24 hours
	if hours >= 24 {
		return fmt.Sprintf("%.0f days", days)
	}

	// More than 60 minutes
	if minutes >= 60 {
		return fmt.Sprintf("%.0f hrs", hours)
	}

	// Less than 60 minutes
	return fmt.Sprintf("%.0f mins", minutes)

}

// UpdatePatientSrv updates patient information with dynamic query generation
func (p *PatientService) UpdatePatientSrv(ctx context.Context, payload dto.UpdatePatientInfo) error {

	// Fetch existing patient
	existingPatient, err := p.PRepo.GetPatientByID(ctx, payload.PatientID, payload.OrganisationID)
	if err != nil {
		return fmt.Errorf("patient not found: %w", err)
	}

	// Initialize dynamic query generation variables
	baseQuery := `UPDATE patients SET `
	var setClauses []string
	var args []interface{}
	argPos := 1

	// Compare and build SET clauses for each field
	if existingPatient.Name != payload.Name && payload.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argPos))
		args = append(args, payload.Name)
		argPos++
	}

	if existingPatient.Age != 0 && strconv.Itoa(existingPatient.Age) != payload.Age && payload.Age != "" {
		age, err := strconv.Atoi(payload.Age)
		if err == nil && age > 0 {
			setClauses = append(setClauses, fmt.Sprintf("age = $%d", argPos))
			args = append(args, age)
			argPos++
		}
	}

	if existingPatient.Gender != payload.Gender && payload.Gender != "" {
		setClauses = append(setClauses, fmt.Sprintf("gender = $%d", argPos))
		args = append(args, payload.Gender)
		argPos++
	}

	if existingPatient.Weight != 0 && payload.Weight != "" {
		weight, err := strconv.ParseFloat(payload.Weight, 64)
		if err == nil && weight > 0 {
			setClauses = append(setClauses, fmt.Sprintf("weight = $%d", argPos))
			args = append(args, int(weight))
			argPos++
		}
	}

	if existingPatient.EmailID != payload.EmailID && payload.EmailID != "" {
		setClauses = append(setClauses, fmt.Sprintf("email_id = $%d", argPos))
		args = append(args, payload.EmailID)
		argPos++
	}

	if existingPatient.MobileNumber != payload.MobileNumber && payload.MobileNumber != "" {
		setClauses = append(setClauses, fmt.Sprintf("mobile_number = $%d", argPos))
		args = append(args, payload.MobileNumber)
		argPos++
	}

	if existingPatient.BloodGroup != payload.BloodGroup && payload.BloodGroup != "" {
		setClauses = append(setClauses, fmt.Sprintf("blood_group = $%d", argPos))
		args = append(args, payload.BloodGroup)
		argPos++
	}

	if existingPatient.Address != payload.Address && payload.Address != "" {
		setClauses = append(setClauses, fmt.Sprintf("address = $%d", argPos))
		args = append(args, payload.Address)
		argPos++
	}

	// Always update the updated_at timestamp
	setClauses = append(setClauses, "updated_at = NOW()")

	// Check if there are any fields to update (besides updated_at)
	if len(setClauses) == 1 {
		return errors.New("no fields updated")
	}

	// Build final query
	baseQuery += strings.Join(setClauses, ", ")
	baseQuery += fmt.Sprintf(` WHERE id = $%d AND organisation_id = $%d`, argPos, argPos+1)

	// Append WHERE clause arguments
	args = append(args, payload.PatientID, payload.OrganisationID)

	// Execute update query
	err = p.PRepo.UpdatePatient(ctx, baseQuery, args)
	if err != nil {
		return fmt.Errorf("failed to update patient: %w", err)
	}

	return nil
}
