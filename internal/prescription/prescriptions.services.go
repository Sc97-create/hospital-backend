package prescription

import (
	"fmt"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/medicine"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/shared/commonfunctions"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PrescriptionService struct {
	DB                      *gorm.DB
	prescriptionRepo        PrescriptionRepositoryInterface
	medicineService         *medicine.MedicineService
	appointmentService      *appointments.AppointmentService
	prescriptionItemService *PrescriptionItemServ
}

func NewPrescriptionService(db *gorm.DB, prescriptionRepo PrescriptionRepositoryInterface, medService *medicine.MedicineService, appointment *appointments.AppointmentService, prescriptionItemServ *PrescriptionItemServ) *PrescriptionService {
	return &PrescriptionService{DB: db, prescriptionRepo: prescriptionRepo, medicineService: medService, appointmentService: appointment, prescriptionItemService: prescriptionItemServ}
}

func (p *PrescriptionService) CreatePrescription(requestdto dto.CreatePrescriptionRequest) (string, error) {
	var prescription Prescription

	appointmentModel, err := p.appointmentService.GetAppntmentByID(requestdto.AppointmentID)
	if err != nil {
		return "", err
	}
	requestdto.PatientID = appointmentModel.PatientID
	prescription = p.createRequest(requestdto)
	tx := p.DB.Begin()
	err = p.prescriptionRepo.CreatePrescription(tx, prescription)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	// with transaction needs to be done
	err = p.prescriptionItemService.AddItems(tx, requestdto.MedicineArray, prescription.ID, prescription.PrescribedBy)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	tx.Commit()

	return prescription.ID, nil
}
func (p *PrescriptionService) createRequest(requestdto dto.CreatePrescriptionRequest) Prescription {
	return Prescription{
		ID:             uuid.NewString(),
		Code:           p.generateCode(),
		Status:         StatusDraft,
		PatientID:      requestdto.PatientID,
		PrescribedBy:   requestdto.PrescribedBy,
		OrganisationID: requestdto.OrganisationID,
		AppointmentID:  requestdto.AppointmentID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
func (p *PrescriptionService) AddPrescriptionItems(payload dto.UpdateRequest) (err error) {
	return p.prescriptionItemService.AddItems(p.DB, payload.MedicineArr, payload.PrescriptionID, payload.UserID)
}
func (p *PrescriptionService) generateCode() string {
	date := time.Now().Format("060102") // YYMMDD
	random := rand.Intn(9000) + 1000
	return fmt.Sprintf("PRX%s%d", date, random)
}
func (p *PrescriptionService) FindMany(limit int, offset int, organisationID string) (prescription []dto.PrescriptionListItem, totalInt int64, err error) {
	skip := commonfunctions.Getskip(limit, offset)
	prescription, err = p.prescriptionRepo.FindMany(limit, skip, organisationID)
	if err != nil {
		return
	}
	totalInt, err = p.prescriptionRepo.Count(organisationID)
	if err != nil {
		return
	}
	return prescription, totalInt, nil
}

func (p *PrescriptionService) mapMedicineNametoID(medicines []medicine.Medicine) map[string]string {
	medicine_map := make(map[string]string)
	for _, each := range medicines {
		medicine_map[each.ID] = each.Name
	}
	return medicine_map
}

func (p *PrescriptionService) tofreqResponse(freq Freq) dto.Freq {
	return dto.Freq{
		Morning:   freq.Morning,
		Afternoon: freq.Afternoon,
		Night:     freq.Night,
	}
}
func (p *PrescriptionService) mapMedicineIDtoName(medMap map[string]string, med []dto.MedicineResponse) []dto.MedicineResponse {
	for i := range med {
		if val, ok := medMap[med[i].MedicineID]; ok {
			med[i].MedicineName = val
		} else {
			med[i].MedicineName = "Unknown"
		}
	}
	return med
}
func (p *PrescriptionService) getMedicineIDS(med []dto.MedicineResponse) []string {
	medids := []string{}
	for _, each := range med {
		medids = append(medids, each.MedicineID)
	}
	return medids
}
func (p *PrescriptionService) UpdateManualStatus(prescriptionID string, appointmentID string) error {
	err := p.DB.Transaction(func(tx *gorm.DB) error {
		err := p.appointmentService.Repository.UpdateStatus(tx, appointments.StatusCompleted, appointmentID)
		if err != nil {
			return err
		}
		err = p.prescriptionRepo.UpdateStatus(tx, StatusSent, prescriptionID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
func (p *PrescriptionService) GetPrescriptionByPatientID(reqmodel dto.PresPatients) (dto.Response, error) {
	dblimit, dbskip := p.parsePagination(reqmodel.Limit, reqmodel.Pageno)
	query := `SELECT
    p.id AS prescription_id,
    p.created_at AS prescription_created_at,
    u.username AS doctor_name,
    COALESCE(
        (
            SELECT json_agg(
                json_build_object(
                    'prescription_item_id', pi.id,
                    'medicine_id', pi.medicine_id,
                    'medicine_name', m.name,
                    'medicine_form', m.form,
                    'medicine_strength', m.strength,
                    'frequency', pi.frequency,
                    'duration_day', pi.duration_day,
                    'duration_type', pi.duration_type,
                    'food_instruction', pi.food_instruction,
                    'quantity', pi.quantity
                ) ORDER BY pi.created_at
            )
            FROM prescription_items pi
            JOIN medicines m ON m.id = pi.medicine_id
            WHERE pi.prescription_id = p.id
        ),
        '[]'::json
    ) AS prescription_item_list
FROM prescriptions p
JOIN users u ON u.id = p.prescribed_by
WHERE p.appointment_id = $1
  AND p.organisation_id = $2
ORDER BY p.created_at ASC
LIMIT $3
OFFSET $4`

	prescriptions, err := p.prescriptionRepo.GetPrescriptionsByAppointmentID(query, reqmodel.OrganisationID, dblimit, dbskip)
	if err != nil {
		return dto.Response{}, err
	}
	presResponse := p.toAppointmentPrescriptionResponse(prescriptions)
	totalCount, err := p.prescriptionRepo.GetPrescriptionByAppointmentIDCount(reqmodel.AppointmentID, reqmodel.OrganisationID)
	if err != nil {
		return dto.Response{}, err
	}
	var response dto.Response
	response.Data = presResponse
	response.Total = int(totalCount)
	response.Code = "200"
	response.Message = "fetched data successfully"
	return response, nil
}

func (p *PrescriptionService) toAppointmentPrescriptionResponse(prescriptions []PrescriptionAppointmentData) []dto.AppointmentPrescriptionResponse {
	responses := make([]dto.AppointmentPrescriptionResponse, 0, len(prescriptions))
	for _, each := range prescriptions {
		items := make([]dto.AppointmentPrescriptionItem, 0, len(each.PrescriptionItemList))
		for _, item := range each.PrescriptionItemList {
			items = append(items, dto.AppointmentPrescriptionItem{
				PrescriptionItemID: item.PrescriptionItemID,
				PrescriptionID:     item.PrescriptionID,
				MedicineID:         item.MedicineID,
				MedicineName:       item.MedicineName,
				MedicineForm:       item.MedicineForm,
				MedicineStrength:   item.MedicineStrength,
				Frequency:          p.tofreqResponse(item.Frequency),
				DurationDay:        item.DurationDay,
				DurationType:       item.DurationType,
				FoodInstruction:    item.FoodInstruction,
				Quantity:           item.Quantity,
			})
		}
		responses = append(responses, dto.AppointmentPrescriptionResponse{
			PrescriptionID:    each.PrescriptionID,
			IssuedAt:          each.PrescriptionCreatedAt,
			DoctorName:        each.DoctorName,
			PrescriptionItems: items,
		})
	}
	return responses
}

func (p *PrescriptionService) parsePagination(limit float64, pageno float64) (int, int) {
	numLimit := int(limit)
	if numLimit <= 0 {
		numLimit = 10
	}
	numpageno := int(pageno)
	if numpageno <= 0 {
		numpageno = 1
	}
	skip := (numpageno - 1) * numLimit
	return numLimit, skip
}
func (p *PrescriptionService) UpdateExtPrescriptionStatus(tx *gorm.DB, prescriptionID string, status string) error {
	var Pstatus Status
	switch status {
	case "fully_dispensed":
		Pstatus = StatusFullyDispensed
	case "partially_dispensed":
		Pstatus = StatusPartiallyDispensed
	default:
		return fmt.Errorf("invalid status")
	}
	err := p.prescriptionRepo.UpdateStatus(tx, Pstatus, prescriptionID)
	if err != nil {
		return err
	}
	return nil
}
