package appointments

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/admins"
	admindto "hospital-backend/internal/admins/dto"
	"hospital-backend/internal/appointments/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultBuffer = 5 * time.Minute

type NotificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

type AppointmentService struct {
	Db                   *gorm.DB
	Repository           AppointmentRepository
	OrganisationSchedule admins.OrganisationScheduleServicer
	NotificationServ     NotificationEnqueuer
}

type validationError struct {
	Field string
	Msg   string
}

func (e *validationError) Error() string {
	return e.Msg
}

func NewAppointmentService(db *gorm.DB, repository AppointmentRepository, organisationSchedule admins.OrganisationScheduleServicer, notificationServ NotificationEnqueuer) *AppointmentService {
	return &AppointmentService{Db: db, Repository: repository, OrganisationSchedule: organisationSchedule, NotificationServ: notificationServ}
}

func (s *AppointmentService) CreateApptmnt(log *zap.Logger, requestPayload dto.NewApptmnt) (resp dto.NewApptmntResp, err error) {
	log = ensureLog(log)

	orgSchedResp, err := s.OrganisationSchedule.GetScheduleByOrganisationID(log, requestPayload.OrganisationID)
	if err != nil || orgSchedResp.ID == "" {
		if err == nil {
			err = wrapError.ErrOrgScheduleNotFound
		}
		log.Warn("appointment create failed",
			zap.String("organisation_id", requestPayload.OrganisationID),
			zap.String("reason", "org_schedule_not_found"),
			zap.Error(err),
		)
		return dto.NewApptmntResp{}, wrapError.ErrOrgScheduleNotFound
	}

	err = s.validateAppointmentFields(requestPayload.StartTime, requestPayload.EndTime, requestPayload.AppointmentDate, requestPayload.PatientID, requestPayload.DoctorID, orgSchedResp.Slotduration)
	if err != nil {
		field := ""
		var ve *validationError
		if errors.As(err, &ve) {
			field = ve.Field
		}
		log.Warn("appointment create failed",
			zap.String("organisation_id", requestPayload.OrganisationID),
			zap.String("patient_id", requestPayload.PatientID),
			zap.String("doctor_id", requestPayload.DoctorID),
			zap.String("reason", "validation"),
			zap.String("field", field),
			zap.Error(err),
		)
		return dto.NewApptmntResp{}, err
	}

	appointmentModel := s.toApptmntModel(requestPayload, orgSchedResp.ID)
	err = s.Repository.Create(log, &appointmentModel)
	if err != nil {
		log.Error("appointment create failed",
			zap.String("organisation_id", requestPayload.OrganisationID),
			zap.String("patient_id", requestPayload.PatientID),
			zap.String("doctor_id", requestPayload.DoctorID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return dto.NewApptmntResp{}, wrapError.ErrAppointmentCreateFailed
	}

	resp.ID = appointmentModel.ID
	resp.Message = AppointmentCreated
	resp.Code = StatusOk

	data, err := s.GetNotificationDetails(log, appointmentModel.ID)
	if err != nil {
		// Appointment already saved — keep success for client; log for ops.
		log.Error("appointment create notification payload failed",
			zap.String("appointment_id", appointmentModel.ID),
			zap.String("organisation_id", requestPayload.OrganisationID),
			zap.String("reason", "notification_payload"),
			zap.Error(err),
		)
		log.Info("appointment create success",
			zap.String("appointment_id", appointmentModel.ID),
			zap.String("appointment_code", appointmentModel.AppointmentCode),
			zap.String("organisation_id", appointmentModel.OrganisationID),
			zap.String("patient_id", appointmentModel.PatientID),
			zap.String("doctor_id", appointmentModel.DoctorID),
			zap.String("created_by", appointmentModel.CreatedBy),
			zap.String("visit_type", appointmentModel.VisitType),
			zap.String("appointment_date", requestPayload.AppointmentDate),
			zap.String("schedule_id", appointmentModel.ScheduleID),
			zap.Bool("notification_enqueued", false),
		)
		return resp, nil
	}

	var notificationRequest notificationdto.CreateRequest
	notificationRequest.Data = data
	notificationRequest.NotificationType = constants.AppointmentCreatedEvent
	notificationRequest.Subject = constants.AppointmentCreateSubject
	ctx := context.Background()
	s.NotificationServ.Create(ctx, notificationRequest)

	log.Info("appointment create success",
		zap.String("appointment_id", appointmentModel.ID),
		zap.String("appointment_code", appointmentModel.AppointmentCode),
		zap.String("organisation_id", appointmentModel.OrganisationID),
		zap.String("patient_id", appointmentModel.PatientID),
		zap.String("doctor_id", appointmentModel.DoctorID),
		zap.String("created_by", appointmentModel.CreatedBy),
		zap.String("visit_type", appointmentModel.VisitType),
		zap.String("appointment_date", requestPayload.AppointmentDate),
		zap.String("schedule_id", appointmentModel.ScheduleID),
		zap.Bool("notification_enqueued", true),
	)
	return
}

func (s *AppointmentService) GetNotificationDetails(log *zap.Logger, appointmentID string) (map[string]interface{}, error) {
	log = ensureLog(log)
	query := `select a.appointment_date,a.start_time,
	a.end_time,a.appointment_code,u.username as doctor_name,p.name as patient_name,
	p.email_id as patient_email_id,p.uh_id as patient_code,p.id as patient_id,a.organisation_id,o.organisation_name as hospital_name
	from appointments a
	join organisations o
	on a.organisation_id=o.id
	join patients p
	on a.patient_id=p.id
	join users u
	on a.doctor_id=u.id
	where a.id = $1`
	notificationData, err := s.Repository.GetNotificationsDetails(log, query, appointmentID)
	if err != nil {
		return nil, err
	}
	return notificationData, nil
}

func (s *AppointmentService) toApptmntModel(reqpayload dto.NewApptmnt, osID string) Appointment {
	var model Appointment
	model.ID = uuid.New().String()
	model.DoctorID = reqpayload.DoctorID
	model.OrganisationID = reqpayload.OrganisationID
	model.PatientID = reqpayload.PatientID
	model.CreatedAt = time.Now()
	model.SeriesID = reqpayload.SeriesID
	model.Status = StatusScheduled
	model.CreatedBy = reqpayload.UserID
	model.VisitType = reqpayload.ReasonForVisit
	model.AppointmentDate, _ = time.Parse(time.DateOnly, reqpayload.AppointmentDate)
	model.StartTime, _ = time.Parse(time.RFC3339, reqpayload.StartTime)
	model.EndTime, _ = time.Parse(time.RFC3339, reqpayload.EndTime)
	model.AppointmentCode = s.generateAppointmentCode()
	model.ScheduleID = osID
	return model
}

func (s *AppointmentService) appendAppointmentDate(appointmentdate string, tstartTime time.Time, tendtime time.Time) (time.Time, time.Time) {
	tAppointmentDate, _ := time.Parse(time.DateOnly, appointmentdate)
	location, _ := time.LoadLocation("Asia/Kolkata")
	dbstarttime := time.Date(tAppointmentDate.Year(), tAppointmentDate.Month(), tAppointmentDate.Day(), tstartTime.Hour(), tstartTime.Minute(), tstartTime.Second(), tstartTime.Nanosecond(), location)
	dbendtime := time.Date(tAppointmentDate.Year(), tAppointmentDate.Month(), tAppointmentDate.Day(), tendtime.Hour(), tendtime.Minute(), tendtime.Second(), tendtime.Nanosecond(), location)
	return dbstarttime, dbendtime
}

func (s *AppointmentService) generateAppointmentCode() string {
	currentDate := time.Now().Format("20060102")
	randomString := uuid.New().String()[:3]
	appointmentCode := fmt.Sprintf("APT-%s-%s", currentDate, randomString)
	return appointmentCode
}

func (s *AppointmentService) validateAppointmentFields(startTime string, endTime string, appointmentDate string, patientID string, doctorID string, slotduration int) (err error) {
	if patientID == "" {
		return &validationError{Field: "patient_id", Msg: "appointment creation failed: patient_id is missing"}
	}
	if doctorID == "" {
		return &validationError{Field: "doctor_id", Msg: "appointment creation failed: doctor_id is missing"}
	}

	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return &validationError{Field: "start_time", Msg: "appointment creation failed: invalid start_time"}
	}
	originalTime := start.Add(time.Duration(slotduration) * time.Minute)

	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return &validationError{Field: "end_time", Msg: "appointment creation failed: invalid end_time"}
	}
	if end.Before(originalTime) {
		return &validationError{Field: "slot_duration", Msg: "something went wrong, please check with administrator"}
	}
	if !start.Before(end) {
		return &validationError{
			Field: "start_time",
			Msg: fmt.Sprintf(
				"appointment creation failed: start_time (%s) must be before end_time (%s)",
				startTime,
				endTime,
			),
		}
	}

	apptDate, err := time.Parse(time.DateOnly, appointmentDate)
	if err != nil {
		return &validationError{Field: "appointment_date", Msg: "appointment creation failed: invalid appointment_date"}
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if apptDate.Before(today) {
		return &validationError{
			Field: "appointment_date",
			Msg: fmt.Sprintf(
				"appointment creation failed: appointment_date (%s) cannot be in the past",
				appointmentDate,
			),
		}
	}
	return nil
}

func (s *AppointmentService) GetSlots(log *zap.Logger, doctorID string, organisationID string, date string) (dto.SlotResponse, error) {
	log = ensureLog(log)
	query := `select id,start_time,end_time from appointments where doctor_id = $1 and organisation_id=$2 and appointment_date=$3`
	appointments, err := s.Repository.GetAppointmentsByIDs(log, query, doctorID, organisationID, date)
	if err != nil {
		log.Error("appointment slots failed",
			zap.String("doctor_id", doctorID),
			zap.String("organisation_id", organisationID),
			zap.String("date", date),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.SlotResponse{}, wrapError.ErrAppointmentSlotsFailed
	}

	orgSchedules, err := s.OrganisationSchedule.GetScheduleByOrganisationID(log, organisationID)
	if err != nil || orgSchedules.ID == "" {
		if err == nil {
			err = wrapError.ErrOrgScheduleNotFound
		}
		log.Warn("appointment slots failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "org_schedule_not_found"),
			zap.Error(err),
		)
		return dto.SlotResponse{}, wrapError.ErrOrgScheduleNotFound
	}

	slots, err := s.checkIfSlotAvailable(appointments, orgSchedules, date)
	if err != nil {
		log.Error("appointment slots failed",
			zap.String("organisation_id", organisationID),
			zap.String("date", date),
			zap.String("reason", "slot_processing"),
			zap.Error(err),
		)
		return dto.SlotResponse{}, wrapError.ErrAppointmentSlotsFailed
	}

	available := 0
	for _, slot := range slots {
		if slot.Allow {
			available++
		}
	}

	var slotResponse dto.SlotResponse
	slotResponse.Apptmnt = s.toSlotResponse(slots)
	slotResponse.Message = SlotFetch
	slotResponse.Code = StatusOk

	log.Info("appointment slots success",
		zap.String("doctor_id", doctorID),
		zap.String("organisation_id", organisationID),
		zap.String("date", date),
		zap.Int("slot_count", len(slots)),
		zap.Int("available_count", available),
	)
	return slotResponse, nil
}

func (s *AppointmentService) toSlotResponse(slot []Slot) []dto.AppointmentSlots {
	var slotResponse []dto.AppointmentSlots
	for _, each := range slot {
		var slotResp dto.AppointmentSlots
		slotResp.StartTime = each.Start
		slotResp.Endtime = each.End
		slotResp.Allow = each.Allow
		slotResponse = append(slotResponse, slotResp)
	}
	return slotResponse
}

func (s *AppointmentService) checkIfSlotAvailable(appointments []Appointment, schedule admindto.GetResponse, date string) ([]Slot, error) {
	allSlots, err := s.createSlots(schedule, date)
	if err != nil {
		return nil, err
	}
	for i, slot := range allSlots {
		if s.isSlotOccupied(slot, appointments) {
			allSlots[i].Allow = false
			continue
		}
	}
	return allSlots, nil
}

func (s *AppointmentService) createSlots(schedule admindto.GetResponse, date string) ([]Slot, error) {
	slotDuration := time.Duration(schedule.Slotduration) * time.Minute
	var slots []Slot
	scheduleStart := s.normalizeTimeOfDay(schedule.Starttime)
	scheduleEnd := s.normalizeTimeOfDay(schedule.Endtime)
	breakstartTime := s.normalizeTimeOfDay(schedule.BreakStarttime)
	breakendtime := s.normalizeTimeOfDay(schedule.BreakEndtime)
	if scheduleEnd.Before(scheduleStart) {
		return nil, errors.New("schedule end time must be after start time")
	}
	currentTime := time.Now()
	currentDate := currentTime.Format(time.DateOnly)
	now := s.normalizeTimeOfDay(currentTime)
	if currentDate == date {
		if !now.Before(scheduleEnd) {
			return []Slot{}, nil
		}
		if now.After(scheduleStart) {
			scheduleStart = now
		}
	}

	for {
		slotEnd := scheduleStart.Add(slotDuration)
		if slotEnd.After(scheduleEnd) {
			break
		}
		if s.timesOverlap(scheduleStart, slotEnd, breakstartTime, breakendtime) {
			scheduleStart = breakendtime.Add(defaultBuffer)
			continue
		}
		dbstarttime, dbendtime := s.appendAppointmentDate(date, scheduleStart, slotEnd)
		slots = append(slots, Slot{Start: dbstarttime, End: dbendtime, Allow: true})
		scheduleStart = slotEnd.Add(defaultBuffer)
	}
	return slots, nil
}

func (s *AppointmentService) isSlotOccupied(slot Slot, appointments []Appointment) bool {
	for _, eachAppointment := range appointments {
		if s.timesOverlap(slot.Start, slot.End, eachAppointment.StartTime, eachAppointment.EndTime) {
			return true
		}
	}
	return false
}

func (s *AppointmentService) normalizeTimeOfDay(value time.Time) time.Time {
	return time.Date(0, 1, 1, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), time.Local)
}

func (s *AppointmentService) timesOverlap(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && startB.Before(endA)
}

func (s *AppointmentService) GetAppointmentsByOrgID(log *zap.Logger, reqModel dto.GetDataReq) ([]dto.AppointmentList, int, error) {
	log = ensureLog(log)
	dblimit, dbpageno := s.parsepagination(reqModel.Limit, reqModel.PageNo)
	reqModel.Dblimit = dblimit
	reqModel.Dbpageno = dbpageno
	reqModel.Search = strings.TrimSpace(reqModel.Search)

	query, args := s.buildQueryWithFilters(reqModel)
	data, err := s.Repository.FindManyByOrganisationID(log, query, args...)
	if err != nil {
		log.Error("appointment list failed",
			zap.String("organisation_id", reqModel.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrAppointmentsFetchFailed
	}
	countQuery, countArgs := s.buildCountQueryWithFilters(reqModel)
	total, err := s.Repository.GetTotalAppointmentsByOrgID(log, countQuery, countArgs...)
	if err != nil {
		log.Error("appointment list failed",
			zap.String("organisation_id", reqModel.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrAppointmentsFetchFailed
	}
	response := s.toAppointmentList(data)
	log.Info("appointment list success",
		zap.String("organisation_id", reqModel.OrganisationID),
		zap.Int("count", len(response)),
		zap.Int("total", total),
	)
	return response, total, nil
}

func (s *AppointmentService) appendAppointmentFilters(query string, reqModel dto.GetDataReq, args []interface{}, argsPos int) (string, []interface{}, int) {
	if reqModel.Date != "" {
		switch reqModel.Date {
		case Today:
			query += `
		AND a.appointment_date >= CURRENT_DATE
		AND a.appointment_date < CURRENT_DATE + INTERVAL '1 day'
	`
		case Tomorrow:
			query += `
		AND a.appointment_date >= CURRENT_DATE + INTERVAL '1 day'
		AND a.appointment_date < CURRENT_DATE + INTERVAL '2 day'`
		case ThisWeek:
			query += `
		AND a.appointment_date >= DATE_TRUNC('week', CURRENT_DATE)
		AND a.appointment_date < DATE_TRUNC('week', CURRENT_DATE) + INTERVAL '7 days'
	`
		case ThisMonth:
			query += `
		AND a.appointment_date >= DATE_TRUNC('month', CURRENT_DATE)
		AND a.appointment_date < DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month'
	`
		}
	}
	if reqModel.DoctorID != "" {
		query += fmt.Sprintf(" AND a.doctor_id = $%d", argsPos)
		args = append(args, reqModel.DoctorID)
		argsPos++
	}
	if reqModel.Status != "" {
		query += fmt.Sprintf(" AND a.status = $%d", argsPos)
		args = append(args, reqModel.Status)
		argsPos++
	}
	if reqModel.VisitType != "" {
		query += fmt.Sprintf(" AND a.visit_type = $%d", argsPos)
		args = append(args, reqModel.VisitType)
		argsPos++
	}
	if reqModel.Search != "" {
		query += fmt.Sprintf(" AND (p.name ILIKE $%d OR a.appointment_code ILIKE $%d)", argsPos, argsPos)
		args = append(args, "%"+reqModel.Search+"%")
		argsPos++
	}
	return query, args, argsPos
}

func (s *AppointmentService) buildQueryWithFilters(reqModel dto.GetDataReq) (string, []interface{}) {
	baseQuery := `
		SELECT
			a.id as appointment_id,
			a.appointment_code,
			a.visit_type,
			a.status,
			a.start_time,
			a.end_time,
			a.appointment_date,
			p.mobile_number,
			p.name as patient_name,
			u2.username AS doctor_name
		FROM appointments a
		JOIN patients p
			ON a.patient_id = p.id
		JOIN users u2
			ON a.doctor_id = u2.id
		WHERE a.organisation_id = $1
		
	`
	args := []interface{}{reqModel.OrganisationID}
	baseQuery, args, argsPos := s.appendAppointmentFilters(baseQuery, reqModel, args, 2)
	baseQuery += " ORDER BY a.appointment_date DESC, a.start_time ASC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argsPos, argsPos+1)
	args = append(args, reqModel.Dblimit, reqModel.Dbpageno)
	return baseQuery, args
}

func (s *AppointmentService) buildCountQueryWithFilters(reqModel dto.GetDataReq) (string, []interface{}) {
	countQuery := `
		SELECT COUNT(*)
		FROM appointments a
		JOIN patients p ON a.patient_id = p.id
		WHERE a.organisation_id = $1
	`
	args := []interface{}{reqModel.OrganisationID}
	countQuery, args, _ = s.appendAppointmentFilters(countQuery, reqModel, args, 2)
	return countQuery, args
}

func (s *AppointmentService) parsepagination(limit float64, pageno float64) (int, int) {
	numLimit := int(limit)
	numpageno := int(pageno)
	skip := 0
	if numpageno != 0 {
		skip = (numpageno - 1) * numLimit
	}
	return numLimit, skip
}

func (s *AppointmentService) toAppointmentList(data []map[string]interface{}) []dto.AppointmentList {
	var response []dto.AppointmentList
	for _, each := range data {
		var singleResp dto.AppointmentList
		singleResp.AppointmentID, _ = each["appointment_id"].(string)
		singleResp.AppointmentCode, _ = each["appointment_code"].(string)
		status, _ := each["status"].(string)
		singleResp.PatientName, _ = each["patient_name"].(string)
		singleResp.DoctorName, _ = each["doctor_name"].(string)
		singleResp.MobileNo, _ = each["mobile_number"].(string)
		starttime, _ := each["start_time"].(time.Time)
		endtime, _ := each["end_time"].(time.Time)
		singleResp.StartTime, singleResp.EndTime = s.formatSEtime(starttime, endtime)
		singleResp.AppointmentDate, _ = each["appointment_date"].(time.Time)
		singleResp.VisitType, _ = each["visit_type"].(string)
		singleResp.Status = string(s.findStatus(status, endtime, singleResp.AppointmentDate))
		response = append(response, singleResp)
	}
	return response
}

func (s *AppointmentService) findStatus(status string, endtime time.Time, appointmentdate time.Time) Status {
	switch status {
	case "ongoing":
		return StatusOngoing
	case "completed":
		return StatusCompleted
	case "cancelled":
		return StatusCancelled
	}
	currenttime := time.Now()
	todayDate := time.Date(currenttime.Year(), currenttime.Month(), currenttime.Day(), 0, 0, 0, 0, time.Local)
	todayendtime := time.Date(currenttime.Year(), currenttime.Month(), currenttime.Day(), endtime.Hour(), endtime.Minute(), endtime.Second(), endtime.Nanosecond(), time.Local)
	if appointmentdate == todayDate {
		if currenttime.After(todayendtime) {
			return StatusReschedule
		}
	}
	if appointmentdate.After(todayDate) {
		return StatusUpcoming
	}
	if appointmentdate.Before(todayDate) {
		return StatusMissed
	}
	return StatusScheduled
}

func (s *AppointmentService) GetAppointmentPreview(log *zap.Logger, organisationID string, appointmentID string) (dto.AppointmentDetails, error) {
	log = ensureLog(log)
	query := `SELECT
    a.appointment_code,
    a.id AS appointment_id,
    a.start_time,
    a.end_time,
	a.visit_type,
	a.status,
    a.appointment_date,
    pa.name,
    pa.age,
	u.username as doctor_name,
    pa.gender,
    pa.mobile_number,
    pr.created_at,
	d.name as department_name,
	os.slot_duration
FROM appointments AS a
JOIN patients AS pa
    ON a.patient_id = pa.id
JOIN users AS u
    ON a.doctor_id = u.id
JOIN departments AS d
    ON u.department_id = d.id
LEFT JOIN prescriptions AS pr
    ON a.id = pr.appointment_id
JOIN organisation_schedules as os
    ON a.schedule_id = os.id
WHERE a.organisation_id = $1 and a.id = $2`
	dbResp, err := s.Repository.GetAppointmentsPreview(log, query, organisationID, appointmentID)
	if err != nil {
		log.Error("appointment preview failed",
			zap.String("organisation_id", organisationID),
			zap.String("appointment_id", appointmentID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.AppointmentDetails{}, wrapError.ErrAppointmentFetchFailed
	}
	if dbResp == nil {
		log.Warn("appointment preview failed",
			zap.String("organisation_id", organisationID),
			zap.String("appointment_id", appointmentID),
			zap.String("reason", "not_found"),
		)
		return dto.AppointmentDetails{}, wrapError.ErrAppointmentNotFound
	}
	id, _ := dbResp["appointment_id"].(string)
	code, _ := dbResp["appointment_code"].(string)
	if id == "" && code == "" {
		log.Warn("appointment preview failed",
			zap.String("organisation_id", organisationID),
			zap.String("appointment_id", appointmentID),
			zap.String("reason", "not_found"),
		)
		return dto.AppointmentDetails{}, wrapError.ErrAppointmentNotFound
	}

	appointmentDetails := s.toAppointmentPreview(dbResp)
	log.Info("appointment preview success",
		zap.String("organisation_id", organisationID),
		zap.String("appointment_id", appointmentDetails.AppointmentID),
		zap.String("appointment_code", appointmentDetails.AppointmentCode),
		zap.String("status", appointmentDetails.Status),
		zap.String("visit_type", appointmentDetails.VisitType),
	)
	return appointmentDetails, nil
}

func (s *AppointmentService) toAppointmentPreview(data map[string]interface{}) dto.AppointmentDetails {
	var response dto.AppointmentDetails
	response.AppointmentID, _ = data["appointment_id"].(string)
	response.AppointmentCode, _ = data["appointment_code"].(string)
	starttime, _ := data["start_time"].(time.Time)
	endtime, _ := data["end_time"].(time.Time)
	response.AppointmentDate, _ = data["appointment_date"].(time.Time)
	response.PatientName, _ = data["name"].(string)
	response.MobileNo, _ = data["mobile_number"].(string)
	response.Notes, _ = data["notes"].(string)
	response.DoctorName, _ = data["doctor_name"].(string)
	response.VisitType, _ = data["visit_type"].(string)
	response.Status, _ = data["status"].(string)
	response.PatientAge, _ = data["age"].(int64)
	response.PatientGender, _ = data["gender"].(string)
	medicines, _ := data["medicines"].([]map[string]interface{})
	response.DepartmentName, _ = data["department_name"].(string)
	response.SlotDuration, _ = data["slot_duration"].(int64)
	response.StartTime, response.EndTime = s.formatSEtime(starttime, endtime)
	response.Status = string(s.findStatus(response.Status, endtime, response.AppointmentDate))
	response.Medicines = len(medicines)
	return response
}

func (s *AppointmentService) GetAppntmentByID(log *zap.Logger, appointmentID string) (Appointment, error) {
	log = ensureLog(log)
	log.Debug("appointment get by id", zap.String("appointment_id", appointmentID))

	appointments, err := s.Repository.GetAppointmentByID(log, appointmentID)
	if err != nil {
		log.Error("appointment get by id failed",
			zap.String("appointment_id", appointmentID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Appointment{}, wrapError.ErrAppointmentFetchFailed
	}
	if appointments.ID == "" {
		log.Warn("appointment get by id failed",
			zap.String("appointment_id", appointmentID),
			zap.String("reason", "not_found"),
		)
		return Appointment{}, wrapError.ErrAppointmentNotFound
	}

	log.Debug("appointment get by id success",
		zap.String("appointment_id", appointments.ID),
		zap.String("organisation_id", appointments.OrganisationID),
		zap.String("patient_id", appointments.PatientID),
		zap.String("status", string(appointments.Status)),
	)
	return appointments, nil
}

func (s *AppointmentService) UpdateStatusInTx(log *zap.Logger, tx *gorm.DB, status string, appointmentID string) error {
	return s.Repository.UpdateStatus(ensureLog(log), tx, status, appointmentID)
}

func (s *AppointmentService) UpdateStatus(log *zap.Logger, updateReq dto.UpdateStatus) (err error) {
	log = ensureLog(log)
	status, err := s.SelectStatus(updateReq.Status)
	if err != nil {
		log.Warn("appointment status update failed",
			zap.String("appointment_id", updateReq.AppointmentID),
			zap.String("status", updateReq.Status),
			zap.String("reason", "invalid_status"),
		)
		return wrapError.ErrInvalidRequest
	}
	err = s.Repository.UpdateStatus(log, s.Db, status, updateReq.AppointmentID)
	if err != nil {
		log.Error("appointment status update failed",
			zap.String("appointment_id", updateReq.AppointmentID),
			zap.String("status", string(status)),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrAppointmentUpdateFailed
	}
	log.Info("appointment status update success",
		zap.String("appointment_id", updateReq.AppointmentID),
		zap.String("status", string(status)),
	)
	return
}

func (s *AppointmentService) SelectStatus(status string) (Status, error) {
	switch status {
	case "completed":
		return StatusCompleted, nil
	case "cancelled":
		return StatusCancelled, nil
	case "scheduled":
		return StatusScheduled, nil
	case "ongoing":
		return StatusOngoing, nil
	default:
		return "", wrapError.ErrInvalidRequest
	}
}

func (s *AppointmentService) GetAppointmentByPatientID(log *zap.Logger, reqModel dto.PatientAppntment) (dto.Response, error) {
	log = ensureLog(log)
	dblimit, dbpageno := s.parsepagination(reqModel.Limit, reqModel.Pageno)
	query, args := s.buidPatientAppntmentFilter(reqModel, dblimit, dbpageno)
	appointments, err := s.Repository.GetAppointmentByPatientID(log, query, args...)
	if err != nil {
		log.Error("appointment patient list failed",
			zap.String("patient_id", reqModel.PatientID),
			zap.String("organisation_id", reqModel.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.Response{}, wrapError.ErrAppointmentsFetchFailed
	}
	appointmentCount, err := s.Repository.GetAppointmentByPatientIDCount(log, reqModel.PatientID, reqModel.OrganisationID)
	if err != nil {
		log.Error("appointment patient list failed",
			zap.String("patient_id", reqModel.PatientID),
			zap.String("organisation_id", reqModel.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return dto.Response{}, wrapError.ErrAppointmentsFetchFailed
	}
	patAppointments := s.toPatientAppntment(appointments)
	var response dto.Response
	response.Data = patAppointments
	response.Total = int(appointmentCount)
	response.Code = "200"
	response.Message = "fetched successfully"
	log.Info("appointment patient list success",
		zap.String("patient_id", reqModel.PatientID),
		zap.String("organisation_id", reqModel.OrganisationID),
		zap.Int("count", len(patAppointments)),
		zap.Int("total", response.Total),
	)
	return response, nil
}

func (s *AppointmentService) toPatientAppntment(appointments []map[string]interface{}) []dto.PatAppointment {
	var patAppointment []dto.PatAppointment
	for _, each := range appointments {
		var eachAppointment dto.PatAppointment
		eachAppointment.AppointmentID, _ = each["appointment_id"].(string)
		eachAppointment.AppointmentCode, _ = each["appointment_code"].(string)
		eachAppointment.AppointmentDate, _ = each["appointment_date"].(time.Time)
		starttime, _ := each["start_time"].(time.Time)
		endtime, _ := each["end_time"].(time.Time)
		eachAppointment.StartTime, _ = s.formatSEtime(starttime, endtime)
		eachAppointment.DepartmentName, _ = each["name"].(string)
		eachAppointment.DoctorName, _ = each["username"].(string)
		eachAppointment.Status, _ = each["status"].(string)
		eachAppointment.VisitType, _ = each["visit_type"].(string)
		eachAppointment.Status = string(s.findStatus(eachAppointment.Status, endtime, eachAppointment.AppointmentDate))
		patAppointment = append(patAppointment, eachAppointment)
	}
	return patAppointment
}

func (s *AppointmentService) buidPatientAppntmentFilter(reqModel dto.PatientAppntment, dblimit int, dbpageno int) (basequery string, args []interface{}) {
	baseQuery := `
	select a.id as appointment_id,
	a.appointment_code,
	a.start_time,
	a.end_time,
	a.appointment_date,
	a.status,
	a.visit_type,
	u.username,
	d.name
	from appointments a
	join users  u
	on a.doctor_id = u.id
	join departments  d
	on u.department_id = d.id
	where patient_id = $1 and a.organisation_id = $2
	`
	args = []interface{}{reqModel.PatientID, reqModel.OrganisationID}
	argsPos := 2
	argsPos++
	if reqModel.Status != "" {
		if strings.EqualFold(reqModel.Status, string(StatusUpcoming)) {
			baseQuery += `and a.appointment_date >=CURRENT_DATE`
		} else {
			baseQuery += fmt.Sprintf("and a.status = $%d", argsPos)
			args = append(args, reqModel.Status)
			argsPos++
		}
	}
	baseQuery += ` order by a.start_time asc`
	baseQuery += fmt.Sprintf(" limit $%d offset $%d", argsPos, argsPos+1)
	args = append(args, dblimit, dbpageno)
	return baseQuery, args
}

func (s *AppointmentService) formatSEtime(start time.Time, end time.Time) (string, string) {
	startimeStr := start.Format("03:04 PM")
	endtimeStr := end.Format("03:04 PM")
	return startimeStr, endtimeStr
}
