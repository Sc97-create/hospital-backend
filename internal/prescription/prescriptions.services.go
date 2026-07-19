package prescription

import (
	"context"
	"fmt"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/medicine"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/notifications/service"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/pkg/constants"
	"hospital-backend/shared/commonfunctions"
	"math/rand"
	"strings"
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
	notificationService     *service.Notificationservice
	orgService              *organisation.OrganisationService
	userService             *employee.EmployeeService
}

func NewPrescriptionService(db *gorm.DB, prescriptionRepo PrescriptionRepositoryInterface, medService *medicine.MedicineService, appointment *appointments.AppointmentService, prescriptionItemServ *PrescriptionItemServ, notificationService *service.Notificationservice, orgService *organisation.OrganisationService, userService *employee.EmployeeService) *PrescriptionService {
	return &PrescriptionService{DB: db, prescriptionRepo: prescriptionRepo, medicineService: medService, appointmentService: appointment, prescriptionItemService: prescriptionItemServ, notificationService: notificationService, orgService: orgService, userService: userService}
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
	//fire and forget
	return prescription.ID, nil
}
func (p *PrescriptionService) getNotificationDetails(prescriptionID string) (map[string]interface{}, error) {
	query := `
SELECT
	p.code AS prescription_code,
	p.created_at AS consulted_on,
	p.patient_id,
	p.organisation_id,
	o.organisation_name AS hospital_name,
	u.username AS doctor_name,
	pa.name AS patient_name,
	pa.email_id AS patient_email_id,
	pa.uh_id AS patient_code,
	COALESCE(
		(
			SELECT json_agg(
				json_build_object(
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
	) AS medicines
FROM prescriptions p
JOIN organisations o ON o.id = p.organisation_id
JOIN users u ON u.id = p.prescribed_by
JOIN patients pa ON pa.id = p.patient_id
WHERE p.id = $1`

	data, err := p.prescriptionRepo.GetNotificationDetails(query, prescriptionID)
	if err != nil {
		return nil, err
	}

	return p.parseNotificationStruct(data), nil
}
func (p *PrescriptionService) parseNotificationStruct(data PrescriptionNotificationData) map[string]interface{} {
	var notificationData = map[string]interface{}{
		"hospital_name":     data.HospitalName,
		"doctor_name":       data.DoctorName,
		"patient_name":      data.PatientName,
		"patient_email_id":  data.PatientEmail,
		"patient_code":      data.PatientCode,
		"consulted_on":      data.ConsultedOn.Format("02 Jan 2006"),
		"prescription_code": data.PrescriptionCode,
		"patient_id":        data.PatientID,
		"organisation_id":   data.OrganisationID,
		"prescribed_by":     data.DoctorName,
	}
	var medicines []map[string]interface{}
	for _, each := range data.Medicines {
		medicines = append(medicines, map[string]interface{}{
			"medicine_name":     each.MedicineName,
			"medicine_form":     each.MedicineForm,
			"medicine_strength": each.MedicineStrength,
			"dosage":            fmt.Sprintf("%.0f-%.0f-%.0f", each.Frequency.Morning, each.Frequency.Afternoon, each.Frequency.Night),
			"duration":          fmt.Sprintf("%.0f %s", each.DurationDay, each.DurationType),
			"quantity":          fmt.Sprintf("%d", each.Quantity),
			"food_instruction":  each.FoodInstruction,
		})
	}
	notificationData["medicines"] = medicines
	return notificationData
}
func (p *PrescriptionService) createRequest(requestdto dto.CreatePrescriptionRequest) Prescription {
	return Prescription{
		ID:             uuid.NewString(),
		Code:           p.generateCode(),
		Status:         constants.StatusDraft,
		PatientID:      requestdto.PatientID,
		PrescribedBy:   requestdto.PrescribedBy,
		OrganisationID: requestdto.OrganisationID,
		AppointmentID:  requestdto.AppointmentID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
func (p *PrescriptionService) AddPrescriptionItems(payload dto.UpdateRequest) (err error) {
	err = p.prescriptionItemService.AddItems(p.DB, payload.MedicineArr, payload.PrescriptionID, payload.UserID)
	if err != nil {
		return err
	}
	return nil
}
func (p *PrescriptionService) generateCode() string {
	date := time.Now().Format("060102") // YYMMDD
	random := rand.Intn(9000) + 1000
	return fmt.Sprintf("PRX%s%d", date, random)
}
func (p *PrescriptionService) FindMany(limit int, offset int, organisationID string, search string) (prescription []dto.PrescriptionListItem, totalInt int64, err error) {
	skip := commonfunctions.Getskip(limit, offset)
	search = strings.TrimSpace(search)

	listQuery := `SELECT p.id, p.code, e.username AS prescribed_by, p.patient_id, p.appointment_id, p.created_at, p.status as status
	FROM prescriptions AS p
	JOIN users AS e ON p.prescribed_by = e.id
	WHERE p.organisation_id = ?`
	listArgs := []interface{}{organisationID}

	countQuery := `SELECT COUNT(*) FROM prescriptions WHERE organisation_id = ?`
	countArgs := []interface{}{organisationID}

	if search != "" {
		listQuery += ` AND p.code ILIKE ?`
		listArgs = append(listArgs, "%"+search+"%")
		countQuery += ` AND code ILIKE ?`
		countArgs = append(countArgs, "%"+search+"%")
	}

	listQuery += ` ORDER BY p.created_at DESC LIMIT ? OFFSET ?`
	listArgs = append(listArgs, limit, skip)

	prescription, err = p.prescriptionRepo.FindMany(listQuery, listArgs...)
	if err != nil {
		return
	}
	if prescription == nil {
		prescription = []dto.PrescriptionListItem{}
	}
	totalInt, err = p.prescriptionRepo.Count(countQuery, countArgs...)
	if err != nil {
		return
	}

	return prescription, totalInt, nil
}

func (p *PrescriptionService) FindByStatus(limit int, offset int, organisationID string, status string) ([]dto.PrescriptionListItem, int64, error) {
	parsedStatus, err := p.parseFilterStatus(status)
	if err != nil {
		return nil, 0, err
	}

	skip := commonfunctions.Getskip(limit, offset)
	prescriptions, err := p.prescriptionRepo.FindByStatus(organisationID, parsedStatus, limit, skip)
	if err != nil {
		return nil, 0, err
	}
	if prescriptions == nil {
		prescriptions = []dto.PrescriptionListItem{}
	}

	total, err := p.prescriptionRepo.CountByStatus(organisationID, parsedStatus)
	if err != nil {
		return nil, 0, err
	}

	return prescriptions, total, nil
}

func (p *PrescriptionService) parseFilterStatus(status string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case constants.StatusDraft:
		return constants.StatusDraft, nil
	case constants.StatusSent:
		return constants.StatusSent, nil
	case constants.StatusPaymentLinkCreated:
		return constants.StatusPaymentLinkCreated, nil
	default:
		return "", fmt.Errorf("status must be %s, %s, or %s", constants.StatusDraft, constants.StatusSent, constants.StatusPaymentLinkCreated)
	}
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
func (p *PrescriptionService) UpdateManualStatus(prescriptionID string, appointmentID string, status string) error {
	//sent
	//cancelled
	err := p.DB.Transaction(func(tx *gorm.DB) error {
		if status == constants.StatusSent {
			err := p.appointmentService.Repository.UpdateStatus(tx, constants.StatusCompleted, appointmentID)
			if err != nil {
				return err
			}
		}

		err := p.prescriptionRepo.UpdateStatus(tx, status, prescriptionID)
		if err != nil {
			return err
		}

		return nil
	})
	//send notification to patient
	if err != nil {
		return err
	}
	if status == constants.StatusSent {
		var notificationRequest notificationdto.CreateRequest
		notifData, err := p.getNotificationDetails(prescriptionID)
		if err != nil {
			return err
		}
		notificationRequest.Data = notifData
		notificationRequest.NotificationType = constants.PrescriptionCreatedEvent
		notificationRequest.Subject = constants.PrescriptionCreatedSubject
		ctx := context.Background()
		p.notificationService.Create(ctx, notificationRequest)
	}

	return nil
}
func (p *PrescriptionService) GetPrescriptionByAppointmentID(reqmodel dto.PresPatients) (dto.Response, error) {
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

	prescriptions, err := p.prescriptionRepo.GetPrescriptionsByAppointmentID(
		query,
		reqmodel.AppointmentID,
		reqmodel.OrganisationID,
		dblimit,
		dbskip,
	)
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

func (p *PrescriptionService) GetPrescriptionsByPatientID(reqmodel dto.PatientPrescriptionsRequest) (dto.Response, error) {
	limit := reqmodel.Limit
	if limit <= 0 {
		limit = 10
	}
	skip := commonfunctions.Getskip(limit, reqmodel.PageNo)

	query := `SELECT
    p.id,
    p.code,
    u.username AS prescribed_by,
    p.patient_id,
    p.appointment_id,
    p.created_at,
    p.status
FROM prescriptions p
JOIN users u ON u.id = p.prescribed_by
WHERE p.patient_id = $1
ORDER BY p.created_at DESC
LIMIT $2
OFFSET $3`

	prescriptions, err := p.prescriptionRepo.GetPrescriptionsByPatientID(
		query,
		reqmodel.PatientID,
		limit,
		skip,
	)
	if err != nil {
		return dto.Response{}, err
	}
	if prescriptions == nil {
		prescriptions = []dto.PrescriptionListItem{}
	}
	totalCount, err := p.prescriptionRepo.GetPrescriptionByPatientIDCount(reqmodel.PatientID)
	if err != nil {
		return dto.Response{}, err
	}

	return dto.Response{
		Data:    prescriptions,
		Code:    "200",
		Message: "prescriptions fetched successfully",
		Total:   int(totalCount),
	}, nil
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
	var Pstatus string
	switch status {
	case constants.StatusFullyDispensed:
		Pstatus = constants.StatusFullyDispensed
	case constants.StatusPartiallyDispensed:
		Pstatus = constants.StatusPartiallyDispensed
	case constants.StatusPaymentLinkCreated:
		Pstatus = constants.StatusPaymentLinkCreated
	default:
		return fmt.Errorf("invalid status")
	}
	err := p.prescriptionRepo.UpdateStatus(tx, Pstatus, prescriptionID)
	if err != nil {
		return err
	}
	return nil
}
