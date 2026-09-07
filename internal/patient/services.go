package patient

import (
	"context"
	"errors"
	"fmt"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NotificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

type PatientService struct {
	PRepo         PatientRepository
	OrgService    organisation.OrganisationServicer
	notifications NotificationEnqueuer
}

type validationError struct {
	Field string
	Msg   string
}

func (e *validationError) Error() string {
	return e.Msg
}

func NewPatientService(p PatientRepository, orgService organisation.OrganisationServicer, notifications NotificationEnqueuer) *PatientService {
	return &PatientService{PRepo: p, OrgService: orgService, notifications: notifications}
}

func (p *PatientService) CreatePatientSrv(log *zap.Logger, payload dto.PatientInfo) (string, error) {
	log = ensureLog(log)

	org, err := p.OrgService.GetOrgByID(log, payload.OrganisationID)
	if err != nil {
		reason := "org_lookup"
		if errors.Is(err, wrapError.ErrOrganisationNotFound) {
			reason = "org_not_found"
		}
		log.Warn("patient create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", reason),
			zap.Error(err),
		)
		return "", wrapError.ErrOrganisationNotFound
	}

	age, weight, err := p.ValidatePatient(payload)
	if err != nil {
		field := ""
		var ve *validationError
		if errors.As(err, &ve) {
			field = ve.Field
		}
		log.Warn("patient create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "validation"),
			zap.String("field", field),
			zap.Error(err),
		)
		return "", err
	}

	patientModel, err := p.ToPatientModel(age, weight, payload)
	if err != nil {
		log.Error("patient create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "model_build"),
			zap.Error(err),
		)
		return "", wrapError.ErrPatientCreateFailed
	}

	err = p.PRepo.Create(log, &patientModel)
	if err != nil {
		if isUniqueViolation(err) {
			log.Error("patient create failed",
				zap.String("organisation_id", payload.OrganisationID),
				zap.String("reason", "duplicate"),
				zap.Error(err),
			)
			return "", wrapError.ErrPatientAlreadyExists
		}
		log.Error("patient create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrPatientCreateFailed
	}

	notificationData, err := p.parseNotificationDetails(patientModel, org)
	if err != nil {
		log.Error("patient create failed",
			zap.String("patient_id", patientModel.ID),
			zap.String("organisation_id", org.ID),
			zap.String("reason", "notification_payload"),
			zap.Error(err),
		)
		return "", wrapError.ErrPatientCreateFailed
	}

	patientModel.OrganisationID = org.ID
	var notificationRequest notificationdto.CreateRequest
	notificationRequest.Data = notificationData
	notificationRequest.NotificationType = constants.PatientCreatedEvent
	notificationRequest.Subject = constants.PatientCreatedSubject
	ctx := context.Background()
	p.notifications.Create(ctx, notificationRequest) // fire and forget

	log.Info("patient create success",
		zap.String("patient_id", patientModel.ID),
		zap.String("uhid", patientModel.UHID),
		zap.String("organisation_id", org.ID),
		zap.String("created_by", payload.UserID),
		zap.Bool("notification_enqueued", true),
	)
	return patientModel.ID, nil
}

func (p *PatientService) parseNotificationDetails(patientModel Patient, orgData organisation.Organisation) (map[string]interface{}, error) {
	return map[string]interface{}{
		"patient_name":     patientModel.Name,
		"patient_email_id": patientModel.EmailID,
		"patient_code":     patientModel.UHID,
		"patient_id":       patientModel.ID,
		"organisation_id":  orgData.ID,
		"hospital_name":    orgData.OrganisationName,
	}, nil
}

func (p *PatientService) ValidatePatient(payload dto.PatientInfo) (int, float64, error) {
	if payload.Name == "" {
		return 0, 0.0, &validationError{Field: "name", Msg: "please provide name"}
	}
	if payload.Gender == "" {
		return 0, 0.0, &validationError{Field: "gender", Msg: "please provide valid gender"}
	}
	age, _ := strconv.Atoi(payload.Age)
	if age < 0 {
		return 0, 0.0, &validationError{Field: "age", Msg: "age should be greater then 0"}
	}
	weight, _ := strconv.ParseFloat(payload.Weight, 64)
	if weight <= 0.0 {
		return 0, 0.0, &validationError{Field: "weight", Msg: "weight should not be 0"}
	}
	return age, weight, nil
}

func (p *PatientService) FindMany(log *zap.Logger, req dto.PatientListReq) (patientResp []dto.PatientResponse, total int64, err error) {
	log = ensureLog(log)
	req.Search = strings.TrimSpace(req.Search)
	req.DBLimit, req.DBOffset = p.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := p.buildPatientListQuery(req)
	patients, err := p.PRepo.ReadMany(log, listQuery, listArgs...)
	if err != nil {
		log.Error("patient list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPatientsFetchFailed
	}

	countQuery, countArgs := p.buildPatientCountQuery(req)
	total, err = p.PRepo.Count(log, countQuery, countArgs...)
	if err != nil {
		log.Error("patient list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPatientsFetchFailed
	}

	patientResp = p.arraymaptopatientResponse(patients)
	log.Info("patient list success",
		zap.String("organisation_id", req.OrganisationID),
		zap.Int("count", len(patientResp)),
		zap.Int64("total", total),
		zap.Bool("has_search", req.Search != ""),
	)
	return
}

func (p *PatientService) buildPatientListQuery(req dto.PatientListReq) (string, []interface{}) {
	baseQuery := `
		SELECT
			id,
			uh_id,
			name,
			gender,
			age,
			weight,
			mobile_number,
			email_id,
			last_visit_date,
			blood_group,
			status,
			created_at,
			address
		FROM patients
		WHERE organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	baseQuery, args, argsPos := p.appendPatientFilters(baseQuery, req, args, 2)
	baseQuery += " ORDER BY created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argsPos, argsPos+1)
	args = append(args, req.DBLimit, req.DBOffset)
	return baseQuery, args
}

func (p *PatientService) buildPatientCountQuery(req dto.PatientListReq) (string, []interface{}) {
	countQuery := `SELECT COUNT(*) FROM patients WHERE organisation_id = $1`
	args := []interface{}{req.OrganisationID}
	countQuery, args, _ = p.appendPatientFilters(countQuery, req, args, 2)
	return countQuery, args
}

func (p *PatientService) appendPatientFilters(query string, req dto.PatientListReq, args []interface{}, argsPos int) (string, []interface{}, int) {
	if req.Search == "" {
		return query, args, argsPos
	}
	query += fmt.Sprintf(
		" AND (name ILIKE $%d OR mobile_number ILIKE $%d OR uh_id ILIKE $%d)",
		argsPos, argsPos, argsPos,
	)
	args = append(args, "%"+req.Search+"%")
	argsPos++
	return query, args, argsPos
}

func (p *PatientService) parsePagination(limit float64, pageNo float64) (int, int) {
	numLimit := int(limit)
	if numLimit <= 0 {
		numLimit = 10
	}
	numPage := int(pageNo)
	if numPage <= 0 {
		numPage = 1
	}
	return numLimit, (numPage - 1) * numLimit
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

func (p *PatientService) FindOne(log *zap.Logger, id string) (pat dto.PatientResponse, err error) {
	log = ensureLog(log)
	patient, err := p.PRepo.ReadOne(log, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("patient get failed",
				zap.String("patient_id", id),
				zap.String("reason", "not_found"),
			)
			return pat, wrapError.ErrPatientNotFound
		}
		log.Error("patient get failed",
			zap.String("patient_id", id),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return pat, wrapError.ErrPatientFetchFailed
	}

	pat = p.maptopatientResponse(patient)
	log.Info("patient get success",
		zap.String("patient_id", patient.ID),
		zap.String("organisation_id", patient.OrganisationID),
		zap.String("uhid", patient.UHID),
	)
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
			PatientID:        each.ID,
			PatientCode:      each.UHID,
			PatientName:      each.Name,
			PatientWeight:    each.Weight,
			PatientGender:    each.Gender,
			PatientPhone:     each.MobileNumber,
			PatientEmail:     each.EmailID,
			PatientAge:       each.Age,
			PatientStatus:    string(each.Status),
			PatientBG:        each.BloodGroup,
			PatientLVD:       each.LastVisitDate,
			PatientAddress:   each.Address,
			PatientCreatedAt: each.CreatedAt,
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

	if days >= 30 {
		return "0"
	}
	if hours >= 24 {
		return fmt.Sprintf("%.0f days", days)
	}
	if minutes >= 60 {
		return fmt.Sprintf("%.0f hrs", hours)
	}
	return fmt.Sprintf("%.0f mins", minutes)
}

func (p *PatientService) GetNotificationPatientByID(log *zap.Logger, patientID string) (map[string]interface{}, error) {
	log = ensureLog(log)
	log.Debug("patient notification lookup", zap.String("patient_id", patientID))

	query := `select p.uh_id as patient_code,p.name as patient_name,p.email_id as patient_email_id,p.mobile_number as patient_phone,p.blood_group as patient_bg,p.address as patient_address,o.organisation_name as hospital_name,p.organisation_id,p.id as patient_id from patients p 
	join organisations o on p.organisation_id=o.id where p.id = $1`
	patient, err := p.PRepo.ReadOneWithOrganisationID(log, query, patientID)
	if err != nil {
		log.Error("patient notification lookup failed",
			zap.String("patient_id", patientID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, err
	}
	if len(patient) == 0 {
		log.Error("patient notification lookup failed",
			zap.String("patient_id", patientID),
			zap.String("reason", "not_found"),
		)
		return nil, wrapError.ErrPatientNotFound
	}

	orgID := ""
	if v, ok := patient["organisation_id"]; ok {
		orgID = fmt.Sprint(v)
	}
	log.Debug("patient notification lookup success",
		zap.String("patient_id", patientID),
		zap.String("organisation_id", orgID),
	)
	return patient, nil
}
