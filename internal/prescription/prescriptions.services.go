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
	// totalInt, err = p.prescriptionRepo.Count(organisationID)
	// if err != nil {
	// 	return
	// }
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
func (p *PrescriptionService) UpdateStatus(prescriptionID string, appointmentID string) error {
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
    p.id,
    p.created_at,
    p.medicines,
    u.username AS doctor_name
FROM prescriptions p
JOIN users u
    ON p.prescribed_by = u.id
WHERE p.patient_id = $1
  AND p.organisation_id = $2
ORDER BY p.created_at ASC
LIMIT $3
OFFSET $4;`
	prescriptions, err := p.prescriptionRepo.GetPrescriptionsByPatientID(query, reqmodel.PatientID, reqmodel.OrganisationID, dblimit, dbskip)
	if err != nil {
		return dto.Response{}, err
	}
	PresResponse := p.toPrescriptionResponse(prescriptions)
	totalCount, err := p.prescriptionRepo.GetPrescriptionByPatientIDCount(reqmodel.PatientID, reqmodel.OrganisationID)
	if err != nil {
		return dto.Response{}, err
	}
	var response dto.Response
	response.Data = PresResponse
	response.Total = int(totalCount)
	response.Code = "200"
	response.Message = "fetched data successfully"
	return response, nil
}
func (p *PrescriptionService) toPrescriptionResponse(Prescription []MixPrescriptionData) []dto.PrescriptionPatientResponse {
	var PrescriptionResponse []dto.PrescriptionPatientResponse
	for _, each := range Prescription {
		var eachPrescription dto.PrescriptionPatientResponse
		eachPrescription.PrescriptionID = each.ID
		eachPrescription.DoctorName = each.DoctorName
		eachPrescription.IssuedAt = each.CreatedAt
		eachPrescription.Medicines = each.Medicines
		eachPrescription.Reason = each.Reason
		PrescriptionResponse = append(PrescriptionResponse, eachPrescription)
	}
	return PrescriptionResponse
}
func (p *PrescriptionService) parsePagination(limit float64, pageno float64) (int, int) {
	numLimit := int(limit)
	numpageno := int(pageno)
	skip := 0
	if numpageno != 0 {
		skip = (numpageno - 1) * numLimit
	}
	return numLimit, skip
}
