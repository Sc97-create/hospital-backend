package prescription

import (
	"context"
	"errors"
	"fmt"
	invoicedto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/medicine"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/pkg/constants"
	"hospital-backend/shared/commonfunctions"
	wrapError "hospital-backend/shared/error"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrescriptionService struct {
	DB                      *gorm.DB
	prescriptionRepo        PrescriptionRepositoryInterface
	appointmentLookup       AppointmentLookup
	appointmentStatus       AppointmentStatusUpdater
	prescriptionItemService PrescriptionItemAdder
	notificationService     NotificationEnqueuer
}

func NewPrescriptionService(
	db *gorm.DB,
	prescriptionRepo PrescriptionRepositoryInterface,
	appointmentLookup AppointmentLookup,
	appointmentStatus AppointmentStatusUpdater,
	prescriptionItemServ PrescriptionItemAdder,
	notificationService NotificationEnqueuer,
) *PrescriptionService {
	return &PrescriptionService{
		DB:                      db,
		prescriptionRepo:        prescriptionRepo,
		appointmentLookup:       appointmentLookup,
		appointmentStatus:       appointmentStatus,
		prescriptionItemService: prescriptionItemServ,
		notificationService:     notificationService,
	}
}

func (p *PrescriptionService) CreatePrescription(log *zap.Logger, requestdto dto.CreatePrescriptionRequest) (string, error) {
	log = ensureLog(log)

	appointmentModel, err := p.appointmentLookup.GetAppntmentByID(log, requestdto.AppointmentID)
	if err != nil {
		reason := "appointment_lookup"
		if errors.Is(err, wrapError.ErrAppointmentNotFound) {
			reason = "appointment_not_found"
			log.Warn("prescription create failed",
				zap.String("appointment_id", requestdto.AppointmentID),
				zap.String("organisation_id", requestdto.OrganisationID),
				zap.String("reason", reason),
				zap.Error(err),
			)
			return "", wrapError.ErrAppointmentNotFound
		}
		log.Error("prescription create failed",
			zap.String("appointment_id", requestdto.AppointmentID),
			zap.String("organisation_id", requestdto.OrganisationID),
			zap.String("reason", reason),
			zap.Error(err),
		)
		return "", wrapError.ErrPrescriptionCreateFailed
	}

	requestdto.PatientID = appointmentModel.PatientID
	prescription := p.createRequest(requestdto)

	tx := p.DB.Begin()
	err = p.prescriptionRepo.CreatePrescription(log, tx, prescription)
	if err != nil {
		tx.Rollback()
		log.Error("prescription create failed",
			zap.String("organisation_id", requestdto.OrganisationID),
			zap.String("appointment_id", requestdto.AppointmentID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrPrescriptionCreateFailed
	}

	err = p.prescriptionItemService.AddItems(log, tx, requestdto.MedicineArray, prescription.ID, prescription.PrescribedBy)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, wrapError.ErrMedicineAlreadyPresent) {
			log.Warn("prescription create failed",
				zap.String("prescription_id", prescription.ID),
				zap.String("organisation_id", requestdto.OrganisationID),
				zap.String("reason", "duplicate_medicine"),
				zap.Error(err),
			)
			return "", wrapError.ErrMedicineAlreadyPresent
		}
		log.Error("prescription create failed",
			zap.String("prescription_id", prescription.ID),
			zap.String("organisation_id", requestdto.OrganisationID),
			zap.String("reason", "db_add_items"),
			zap.Error(err),
		)
		return "", wrapError.ErrPrescriptionCreateFailed
	}

	if err = tx.Commit().Error; err != nil {
		log.Error("prescription create failed",
			zap.String("prescription_id", prescription.ID),
			zap.String("organisation_id", requestdto.OrganisationID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return "", wrapError.ErrPrescriptionCreateFailed
	}

	log.Info("prescription create success",
		zap.String("prescription_id", prescription.ID),
		zap.String("prescription_code", prescription.Code),
		zap.String("organisation_id", prescription.OrganisationID),
		zap.String("patient_id", prescription.PatientID),
		zap.String("appointment_id", prescription.AppointmentID),
		zap.String("prescribed_by", prescription.PrescribedBy),
		zap.Int("item_count", len(requestdto.MedicineArray)),
		zap.String("status", prescription.Status),
	)
	return prescription.ID, nil
}

func (p *PrescriptionService) getNotificationDetails(log *zap.Logger, prescriptionID string) (map[string]interface{}, error) {
	log = ensureLog(log)
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

	data, err := p.prescriptionRepo.GetNotificationDetails(log, query, prescriptionID)
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

func (p *PrescriptionService) AddPrescriptionItems(log *zap.Logger, payload dto.UpdateRequest) (err error) {
	log = ensureLog(log)
	err = p.prescriptionItemService.AddItems(log, p.DB, payload.MedicineArr, payload.PrescriptionID, payload.UserID)
	if err != nil {
		if errors.Is(err, wrapError.ErrMedicineAlreadyPresent) {
			log.Warn("prescription items add failed",
				zap.String("prescription_id", payload.PrescriptionID),
				zap.String("reason", "duplicate_medicine"),
				zap.Error(err),
			)
			return wrapError.ErrMedicineAlreadyPresent
		}
		log.Error("prescription items add failed",
			zap.String("prescription_id", payload.PrescriptionID),
			zap.String("reason", "db_add_items"),
			zap.Error(err),
		)
		return wrapError.ErrPrescriptionUpdateFailed
	}
	log.Info("prescription items add success",
		zap.String("prescription_id", payload.PrescriptionID),
		zap.Int("item_count", len(payload.MedicineArr)),
		zap.String("prescribed_by", payload.UserID),
	)
	return nil
}

func (p *PrescriptionService) generateCode() string {
	date := time.Now().Format("060102") // YYMMDD
	random := rand.Intn(9000) + 1000
	return fmt.Sprintf("PRX%s%d", date, random)
}

func (p *PrescriptionService) FindMany(log *zap.Logger, req dto.FindManyRequest) (prescription []dto.PrescriptionListItem, totalInt int64, err error) {
	log = ensureLog(log)
	req.Search = strings.TrimSpace(req.Search)
	req.Status = strings.TrimSpace(req.Status)
	req.DBLimit, req.DBOffset = p.parsePagination(req.Limit, req.PageNo)

	if req.Status != "" {
		parsedStatus, statusErr := p.parseFilterStatus(req.Status)
		if statusErr != nil {
			log.Warn("prescription list failed",
				zap.String("organisation_id", req.OrganisationID),
				zap.String("status", req.Status),
				zap.String("reason", "invalid_status"),
			)
			return nil, 0, wrapError.ErrInvalidRequest
		}
		req.Status = parsedStatus
	}

	listQuery, listArgs := p.buildPrescriptionListQuery(req)
	prescription, err = p.prescriptionRepo.FindMany(log, listQuery, listArgs...)
	if err != nil {
		log.Error("prescription list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPrescriptionsFetchFailed
	}
	if prescription == nil {
		prescription = []dto.PrescriptionListItem{}
	}

	countQuery, countArgs := p.buildPrescriptionCountQuery(req)
	totalInt, err = p.prescriptionRepo.Count(log, countQuery, countArgs...)
	if err != nil {
		log.Error("prescription list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPrescriptionsFetchFailed
	}

	log.Info("prescription list success",
		zap.String("organisation_id", req.OrganisationID),
		zap.Int("count", len(prescription)),
		zap.Int64("total", totalInt),
		zap.Bool("has_search", req.Search != ""),
		zap.String("status", req.Status),
	)
	return prescription, totalInt, nil
}

func (p *PrescriptionService) buildPrescriptionListQuery(req dto.FindManyRequest) (string, []interface{}) {
	baseQuery := `
		SELECT
			p.id,
			p.code,
			e.username AS prescribed_by,
			p.patient_id,
			pt.name AS patient_name,
			p.appointment_id,
			p.created_at,
			p.status AS status
		FROM prescriptions AS p
		JOIN users AS e ON p.prescribed_by = e.id
		JOIN patients AS pt ON p.patient_id = pt.id
		WHERE p.organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	baseQuery, args, argsPos := p.appendPrescriptionFilters(baseQuery, req, args, 2)
	baseQuery += " ORDER BY p.created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argsPos, argsPos+1)
	args = append(args, req.DBLimit, req.DBOffset)
	return baseQuery, args
}

func (p *PrescriptionService) buildPrescriptionCountQuery(req dto.FindManyRequest) (string, []interface{}) {
	countQuery := `
		SELECT COUNT(*)
		FROM prescriptions AS p
		JOIN patients AS pt ON p.patient_id = pt.id
		WHERE p.organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	countQuery, args, _ = p.appendPrescriptionFilters(countQuery, req, args, 2)
	return countQuery, args
}

func (p *PrescriptionService) appendPrescriptionFilters(query string, req dto.FindManyRequest, args []interface{}, argsPos int) (string, []interface{}, int) {
	if req.Status != "" {
		query += fmt.Sprintf(" AND p.status = $%d", argsPos)
		args = append(args, req.Status)
		argsPos++
	}
	if req.Search != "" {
		query += fmt.Sprintf(" AND (p.code ILIKE $%d OR pt.name ILIKE $%d)", argsPos, argsPos)
		args = append(args, "%"+req.Search+"%")
		argsPos++
	}
	return query, args, argsPos
}

func (p *PrescriptionService) FindByStatus(log *zap.Logger, limit int, offset int, organisationID string, status string) ([]dto.PrescriptionListItem, int64, error) {
	log = ensureLog(log)
	parsedStatus, err := p.parseFilterStatus(status)
	if err != nil {
		log.Warn("prescription status list failed",
			zap.String("organisation_id", organisationID),
			zap.String("status", status),
			zap.String("reason", "invalid_status"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrInvalidRequest
	}

	skip := commonfunctions.Getskip(limit, offset)
	prescriptions, err := p.prescriptionRepo.FindByStatus(log, organisationID, parsedStatus, limit, skip)
	if err != nil {
		log.Error("prescription status list failed",
			zap.String("organisation_id", organisationID),
			zap.String("status", parsedStatus),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPrescriptionsFetchFailed
	}
	if prescriptions == nil {
		prescriptions = []dto.PrescriptionListItem{}
	}

	total, err := p.prescriptionRepo.CountByStatus(log, organisationID, parsedStatus)
	if err != nil {
		log.Error("prescription status list failed",
			zap.String("organisation_id", organisationID),
			zap.String("status", parsedStatus),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPrescriptionsFetchFailed
	}

	log.Info("prescription status list success",
		zap.String("organisation_id", organisationID),
		zap.String("status", parsedStatus),
		zap.Int("count", len(prescriptions)),
		zap.Int64("total", total),
	)
	return prescriptions, total, nil
}

func (p *PrescriptionService) parseFilterStatus(status string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case constants.StatusDraft:
		return constants.StatusDraft, nil
	case constants.StatusSent:
		return constants.StatusSent, nil
	case constants.StatusPaymentPending, constants.StatusPaymentLinkCreated:
		return constants.StatusPaymentPending, nil
	case constants.StatusCompleted:
		return constants.StatusCompleted, nil
	case constants.StatusTentative:
		return constants.StatusTentative, nil
	default:
		return "", fmt.Errorf(
			"status must be %s, %s, %s, %s, or %s",
			constants.StatusDraft,
			constants.StatusSent,
			constants.StatusPaymentPending,
			constants.StatusCompleted,
			constants.StatusTentative,
		)
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

func (p *PrescriptionService) UpdateManualStatus(log *zap.Logger, prescriptionID string, appointmentID string, status string) error {
	log = ensureLog(log)

	if _, err := p.parseManualStatus(status); err != nil {
		log.Warn("prescription status update failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("status", status),
			zap.String("reason", "invalid_status"),
		)
		return wrapError.ErrInvalidRequest
	}

	if status == constants.StatusSent && appointmentID == "" {
		log.Warn("prescription status update failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("status", status),
			zap.String("reason", "missing_appointment_id"),
		)
		return wrapError.ErrInvalidRequest
	}

	err := p.DB.Transaction(func(tx *gorm.DB) error {
		if status == constants.StatusSent {
			err := p.appointmentStatus.UpdateStatusInTx(log, tx, constants.StatusCompleted, appointmentID)
			if err != nil {
				log.Error("prescription status update failed",
					zap.String("prescription_id", prescriptionID),
					zap.String("appointment_id", appointmentID),
					zap.String("reason", "appointment_status"),
					zap.Error(err),
				)
				return wrapError.ErrPrescriptionUpdateFailed
			}
		}

		err := p.prescriptionRepo.UpdateStatus(log, tx, status, prescriptionID)
		if err != nil {
			log.Error("prescription status update failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("status", status),
				zap.String("reason", "db_update"),
				zap.Error(err),
			)
			return wrapError.ErrPrescriptionUpdateFailed
		}
		return nil
	})
	if err != nil {
		return err
	}

	notificationEnqueued := false
	if status == constants.StatusSent {
		notifData, notifyErr := p.getNotificationDetails(log, prescriptionID)
		if notifyErr != nil {
			log.Error("prescription status notify failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("reason", "notification_payload"),
				zap.Error(notifyErr),
			)
		} else {
			var notificationRequest notificationdto.CreateRequest
			notificationRequest.Data = notifData
			notificationRequest.NotificationType = constants.PrescriptionCreatedEvent
			notificationRequest.Subject = constants.PrescriptionCreatedSubject
			ctx := context.Background()
			if p.notificationService != nil {
				_ = p.notificationService.Create(ctx, notificationRequest)
				notificationEnqueued = true
			}
		}
	}

	log.Info("prescription status update success",
		zap.String("prescription_id", prescriptionID),
		zap.String("status", status),
		zap.String("appointment_id", appointmentID),
		zap.Bool("notification_enqueued", notificationEnqueued),
	)
	return nil
}

func (p *PrescriptionService) parseManualStatus(status string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case constants.StatusDraft:
		return constants.StatusDraft, nil
	case constants.StatusSent:
		return constants.StatusSent, nil
	case constants.StatusPaymentPending, constants.StatusPaymentLinkCreated:
		return constants.StatusPaymentPending, nil
	case constants.StatusCompleted:
		return constants.StatusCompleted, nil
	case constants.StatusTentative:
		return constants.StatusTentative, nil
	case constants.StatusCancelled:
		return constants.StatusCancelled, nil
	default:
		return "", wrapError.ErrInvalidRequest
	}
}

func (p *PrescriptionService) GetPrescriptionByAppointmentID(log *zap.Logger, reqmodel dto.PresPatients) (dto.Response, error) {
	log = ensureLog(log)
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
		log,
		query,
		reqmodel.AppointmentID,
		reqmodel.OrganisationID,
		dblimit,
		dbskip,
	)
	if err != nil {
		log.Error("prescription appointment list failed",
			zap.String("appointment_id", reqmodel.AppointmentID),
			zap.String("organisation_id", reqmodel.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.Response{}, wrapError.ErrPrescriptionsFetchFailed
	}
	presResponse := p.toAppointmentPrescriptionResponse(prescriptions)
	totalCount, err := p.prescriptionRepo.GetPrescriptionByAppointmentIDCount(log, reqmodel.AppointmentID, reqmodel.OrganisationID)
	if err != nil {
		log.Error("prescription appointment list failed",
			zap.String("appointment_id", reqmodel.AppointmentID),
			zap.String("organisation_id", reqmodel.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return dto.Response{}, wrapError.ErrPrescriptionsFetchFailed
	}
	var response dto.Response
	response.Data = presResponse
	response.Total = int(totalCount)
	response.Code = "200"
	response.Message = "fetched data successfully"
	log.Info("prescription appointment list success",
		zap.String("appointment_id", reqmodel.AppointmentID),
		zap.String("organisation_id", reqmodel.OrganisationID),
		zap.Int("count", len(presResponse)),
		zap.Int("total", response.Total),
	)
	return response, nil
}

func (p *PrescriptionService) GetPrescriptionsByPatientID(log *zap.Logger, reqmodel dto.PatientPrescriptionsRequest) (dto.Response, error) {
	log = ensureLog(log)
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
		log,
		query,
		reqmodel.PatientID,
		limit,
		skip,
	)
	if err != nil {
		log.Error("prescription patient list failed",
			zap.String("patient_id", reqmodel.PatientID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.Response{}, wrapError.ErrPrescriptionsFetchFailed
	}
	if prescriptions == nil {
		prescriptions = []dto.PrescriptionListItem{}
	}
	totalCount, err := p.prescriptionRepo.GetPrescriptionByPatientIDCount(log, reqmodel.PatientID)
	if err != nil {
		log.Error("prescription patient list failed",
			zap.String("patient_id", reqmodel.PatientID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return dto.Response{}, wrapError.ErrPrescriptionsFetchFailed
	}

	log.Info("prescription patient list success",
		zap.String("patient_id", reqmodel.PatientID),
		zap.Int("count", len(prescriptions)),
		zap.Int("total", int(totalCount)),
	)
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

func (p *PrescriptionService) UpdateExtPrescriptionStatus(log *zap.Logger, tx *gorm.DB, prescriptionID string, status string) error {
	log = ensureLog(log)
	var Pstatus string
	switch status {
	case constants.StatusPaymentPending, constants.StatusPaymentLinkCreated:
		Pstatus = constants.StatusPaymentPending
	case constants.StatusCompleted:
		Pstatus = constants.StatusCompleted
	case constants.StatusTentative:
		Pstatus = constants.StatusTentative
	case constants.StatusSent:
		Pstatus = constants.StatusSent
	default:
		log.Warn("prescription external status update failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("status", status),
			zap.String("reason", "invalid_status"),
		)
		return wrapError.ErrInvalidRequest
	}
	err := p.prescriptionRepo.UpdateStatus(log, tx, Pstatus, prescriptionID)
	if err != nil {
		log.Error("prescription external status update failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("status", Pstatus),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrPrescriptionUpdateFailed
	}
	log.Info("prescription external status update success",
		zap.String("prescription_id", prescriptionID),
		zap.String("status", Pstatus),
	)
	return nil
}

// ResolveAndUpdateParentStatus sets parent status from all items:
// completed if every item is fully dispensed; otherwise tentative.
func (p *PrescriptionService) ResolveAndUpdateParentStatus(log *zap.Logger, tx *gorm.DB, prescriptionID string, invoiceItems []invoicedto.MedInvoiceItemResponse) error {
	log = ensureLog(log)
	partiallyDispensed := 0
	for _, each := range invoiceItems {
		if each.PrescriptionItemStatus != constants.StatusFullyDispensed {
			partiallyDispensed++
		}
	}
	status := constants.StatusCompleted
	if partiallyDispensed > 0 {
		status = constants.StatusTentative
	}
	log.Info("prescription parent status resolved",
		zap.String("prescription_id", prescriptionID),
		zap.String("status", status),
		zap.Int("partial_item_count", partiallyDispensed),
	)
	return p.UpdateExtPrescriptionStatus(log, tx, prescriptionID, status)
}
