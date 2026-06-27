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
	DB                 *gorm.DB
	prescriptionRepo   PrescriptionRepositoryInterface
	medicineService    *medicine.MedicineService
	appointmentService *appointments.AppointmentService
}

func NewPrescriptionService(db *gorm.DB, prescriptionRepo PrescriptionRepositoryInterface, medService *medicine.MedicineService, appointment *appointments.AppointmentService) *PrescriptionService {
	return &PrescriptionService{DB: db, prescriptionRepo: prescriptionRepo, medicineService: medService, appointmentService: appointment}
}

func (p *PrescriptionService) CreatePrescription(requestdto dto.CreatePrescriptionRequest) (string, error) {
	var prescription Prescription

	appointmentModel, err := p.appointmentService.GetAppntmentByID(requestdto.AppointmentID)
	if err != nil {
		return "", err
	}
	requestdto.PatientID = appointmentModel.PatientID
	prescription = p.createRequest(requestdto)
	err = p.prescriptionRepo.CreatePrescription(prescription)
	if err != nil {
		return "", err
	}

	return prescription.ID, nil
}
func (p *PrescriptionService) createRequest(requestdto dto.CreatePrescriptionRequest) Prescription {
	medicines := p.toMedicineList(requestdto.MedicineArray)
	return Prescription{
		ID:             uuid.NewString(),
		Code:           p.generateCode(),
		Status:         StatusDraft,
		PatientID:      requestdto.PatientID,
		PrescribedBy:   requestdto.PrescribedBy,
		OrganisationID: requestdto.OrganisationID,
		AppointmentID:  requestdto.AppointmentID,
		Medicines:      medicines,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
func (p *PrescriptionService) toMedicineList(medicine []dto.MedicineArray) []Medicines {
	var medicines MedicineList
	//need to calculate quantity for each medicine
	for _, each := range medicine {
		var freq Freq
		freq.Morning = each.Morning
		freq.Afternoon = each.Afternoon
		freq.Night = each.Night
		medicines = append(medicines, Medicines{
			MedicineID:      each.MedicineID,
			DurationDay:     each.DurationDay,
			DurationType:    each.DurationType,
			Quantity:        each.Quantity,
			MedicineType:    each.MedicineType,
			FoodInstruction: each.FoodInstruction,
			Frequency:       freq,
			Dosage:          each.Dosage,
		})
	}
	return medicines
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
func (p *PrescriptionService) UpdatePrescription(requestdto dto.UpdateRequest) error {

	query := `SELECT medicines,id FROM prescriptions WHERE id = $1`
	presc, err := p.prescriptionRepo.FindPrescriptionByID(query, requestdto.PrescriptionID)
	if err != nil {
		return err
	}
	medArr := p.appendtoexistingarray(presc.Medicines, requestdto.MedicineArr)
	var updatePrescription Prescription
	updatePrescription.Medicines = medArr
	updatePrescription.UpdatedAt = time.Now()
	updatePrescription.ID = requestdto.PrescriptionID
	err = p.prescriptionRepo.UpdatePrescription(updatePrescription)
	if err != nil {
		return err
	}

	return nil
}
func (p *PrescriptionService) appendtoexistingarray(medicinearr MedicineList, newMedicine []dto.MedicineArray) MedicineList {
	medicine := p.toMedicineList(newMedicine)
	medicinearr = append(medicinearr, medicine...)
	return medicinearr
}
func (p *PrescriptionService) FindPrescriptionByID(id string, limit int, offset int) ([]dto.MedicineResponse, int, time.Time, error) {
	query := `SELECT id,medicines,created_at FROM prescriptions WHERE id = $1`
	prescription, err := p.prescriptionRepo.FindPrescriptionByID(query, id)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	totalCount := len(prescription.Medicines)

	// Apply pagination to the medicines slice
	start := commonfunctions.Getskip(limit, offset)
	if start > totalCount {
		start = totalCount
	}
	end := start + limit
	if end > totalCount || limit == 0 {
		end = totalCount
	}

	paginatedMedicines := prescription.Medicines[start:end]

	medicinelist := p.getMedicines(paginatedMedicines)
	medids := p.getMedicineIDS(medicinelist)
	medicines, err := p.medicineService.FindNamesByIds(medids)
	if err != nil {
		return nil, 0, prescription.CreatedAt, err
	}
	medicinemap := p.mapMedicineNametoID(medicines)
	medicinelist = p.mapMedicineIDtoName(medicinemap, medicinelist)

	return medicinelist, totalCount, prescription.CreatedAt, nil
}
func (p *PrescriptionService) mapMedicineNametoID(medicines []medicine.Medicine) map[string]string {
	medicine_map := make(map[string]string)
	for _, each := range medicines {
		medicine_map[each.ID] = each.Name
	}
	return medicine_map
}
func (p *PrescriptionService) getMedicines(med MedicineList) []dto.MedicineResponse {
	var medicines []dto.MedicineResponse
	for _, each := range med {
		freq := p.tofreqResponse(each.Frequency)
		medicines = append(medicines, dto.MedicineResponse{
			MedicineID:      each.MedicineID,
			MedicineName:    each.MedicineName,
			Frequency:       freq,
			Quantity:        each.Quantity,
			DurationDay:     each.DurationDay,
			DurationType:    each.DurationType,
			TabletForm:      each.TabletForm,
			FoodInstruction: each.FoodInstruction,
			MedicineType:    each.MedicineType,
			Dosage:          each.Dosage,
		})
	}
	return medicines
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
	query := `select p.id, p.created,p.medicines,u.username as doctor_name from prescriptions p 
	join users u
	on  p.prescribed_by = u.id
	where patient_id = $1 and organisation_id = $2
	order by created_at asc
	limit = $3
	offset = $4`
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
func (p *PrescriptionService) toPrescriptionResponse(Prescription []map[string]interface{}) []dto.PrescriptionPatientResponse {
	var PrescriptionResponse []dto.PrescriptionPatientResponse
	for _, each := range Prescription {
		var eachPrescription dto.PrescriptionPatientResponse
		eachPrescription.DoctorName, _ = each["doctor_name"].(string)
		eachPrescription.IssuedAt, _ = each["created_at"].(time.Time)
		eachPrescription.Medicines, _ = each["medicines"].(MedicineList)
		eachPrescription.Reason, _ = each["reason"].(string)
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
