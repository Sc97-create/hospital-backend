package appointments

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/admins"
	admindto "hospital-backend/internal/admins/dto"
	"hospital-backend/internal/appointments/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/notifications/service"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultBuffer = 5 * time.Minute

// find time slots for the appointment
// create appointment
// list appointments for a patient
// list appointments for a doctor
// update appointment
// delete appointment

type AppointmentService struct {
	Db                   *gorm.DB
	Repository           AppointmentRepository
	OrganisationSchedule *admins.OrganisationScheduleService
	NotificationServ     *service.Notificationservice
}

func NewAppointmentService(db *gorm.DB, repository AppointmentRepository, organisationSchedule *admins.OrganisationScheduleService, notificationServ *service.Notificationservice) *AppointmentService {
	return &AppointmentService{Db: db, Repository: repository, OrganisationSchedule: organisationSchedule, NotificationServ: notificationServ}
}

// delete appointments handle seperately
func (s *AppointmentService) CreateApptmnt(requestPayload dto.NewApptmnt) (resp dto.NewApptmntResp, err error) {
	orgSchedResp, err := s.OrganisationSchedule.GetScheduleByOrganisationID(requestPayload.OrganisationID)
	if err != nil {
		return dto.NewApptmntResp{}, err
	}
	//get appointment by patientID if >1 then follow up
	err = s.validateAppointmentFields(requestPayload.StartTime, requestPayload.EndTime, requestPayload.AppointmentDate, requestPayload.PatientID, requestPayload.DoctorID, orgSchedResp.Slotduration)
	if err != nil {
		return dto.NewApptmntResp{}, err
	}

	appointmentModel := s.toApptmntModel(requestPayload, orgSchedResp.ID)
	err = s.Repository.Create(&appointmentModel)
	if err != nil {
		return
	}
	resp.ID = appointmentModel.ID
	resp.Message = AppointmentCreated
	resp.Code = StatusOk
	data, err := s.GetNotificationDetails(appointmentModel.ID)
	if err != nil {
		return
	}
	var notificationRequest notificationdto.CreateRequest
	notificationRequest.Data = data
	notificationRequest.NotificationType = AppointmentCreatedEvent
	ctx := context.Background()
	s.NotificationServ.Create(ctx, notificationRequest)
	return
}
func (s *AppointmentService) GetNotificationDetails(appointmentID string) (map[string]interface{}, error) {
	query := `select a.appointment_date,a.start_time,
	a.end_time,a.appointment_code,u.username as doctor_name,p.name as patient_name,
	p.email_id as patient_email_id,p.uh_id as patient_code,p.id as patient_id,a.organisation_id,o.hospital_name
	from appointments a
	join organisations o
	on a.organisation_id=o.id
	join patients p
	on a.patient_id=p.id
	join users u
	on a.doctor_id=u.id
	where a.id = $1`
	notificationData, err := s.Repository.GetNotificationsDetails(query, appointmentID)
	if err != nil {
		return nil, err
	}
	return notificationData, nil
}

// func (s *AppointmentService) formatNotificationData(data map[string]interface{}) dto.NotificationModel {
// 	var notificationData dto.NotificationModel

// 	appointmentDate, _ := data["appointment_date"].(time.Time)
// 	startTime, _ := data["start_time"].(time.Time)
// 	endTime, _ := data["end_time"].(time.Time)

// 	notificationData.AppointmentCode, _ = data["appointment_code"].(string)
// 	notificationData.AppointmentDate = appointmentDate.Format("02 Jan 2006")
// 	notificationData.AppointmentTime = fmt.Sprintf("%s - %s", startTime.Format("03:04 PM"), endTime.Format("03:04 PM"))
// 	notificationData.DoctorName, _ = data["doctor_name"].(string)
// 	notificationData.HospitalName, _ = data["hospital_name"].(string)
// 	notificationData.PatientName, _ = data["patient_name"].(string)

// 	return notificationData

// }
func (s *AppointmentService) toApptmntModel(reqpayload dto.NewApptmnt, osID string) Appointment {
	var model Appointment
	model.ID = uuid.New().String()
	model.DoctorID = reqpayload.DoctorID
	model.OrganisationID = reqpayload.OrganisationID
	model.PatientID = reqpayload.PatientID
	model.CreatedAt = time.Now()
	model.SeriesID = reqpayload.SeriesID
	model.Status = StatusScheduled
	model.StartTime, _ = time.Parse(time.RFC3339, reqpayload.StartTime)
	model.EndTime, _ = time.Parse(time.RFC3339, reqpayload.EndTime)
	model.CreatedBy = reqpayload.UserID
	model.AppointmentDate, _ = time.Parse(time.DateOnly, reqpayload.AppointmentDate)
	model.VisitType = reqpayload.ReasonForVisit
	model.AppointmentCode = s.generateAppointmentCode()
	model.ScheduleID = osID
	return model

}
func (s *AppointmentService) generateAppointmentCode() string {
	currentDate := time.Now().Format("20060102")
	randomString := uuid.New().String()[:3]
	appointmentCode := fmt.Sprintf("APT-%s-%s", currentDate, randomString)
	return appointmentCode
}
func (s *AppointmentService) validateAppointmentFields(startTime string, endTime string, appointmentDate string, patientID string, doctorID string, slotduration int) (err error) {

	if patientID == "" {
		return fmt.Errorf("appointment creation failed: patient_id is missing")
	}

	if doctorID == "" {
		return fmt.Errorf("appointment creation failed: doctor_id is missing")
	}

	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return fmt.Errorf("appointment creation failed: invalid start_time")
	}
	originalTime := start.Add(time.Duration(time.Duration(slotduration).Minutes()))

	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return fmt.Errorf("appointment creation failed: invalid end_time")
	}
	if end.Before(originalTime) {
		return fmt.Errorf("something went wrong, please check with administrator")
	}

	if !start.Before(end) {
		return fmt.Errorf(
			"appointment creation failed: start_time (%s) must be before end_time (%s)",
			startTime,
			endTime,
		)
	}

	apptDate, err := time.Parse(time.DateOnly, appointmentDate)
	if err != nil {
		return fmt.Errorf("appointment creation failed: invalid appointment_date")
	}

	now := time.Now()
	today := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		now.Location(),
	)

	if apptDate.Before(today) {
		return fmt.Errorf(
			"appointment creation failed: appointment_date (%s) cannot be in the past",
			appointmentDate,
		)
	}

	return nil

}

func (s *AppointmentService) GetSlots(doctorID string, organisationID string, date string) (dto.SlotResponse, error) {
	query := `select id,start_time,end_time from appointments where doctor_id = $1 and organisation_id=$2 and appointment_date=$3`
	appointments, err := s.Repository.GetAppointmentsByIDs(query, doctorID, organisationID, date)
	if err != nil {
		return dto.SlotResponse{}, errors.New("failed to fetch data from db")
	}

	orgSchedules, err := s.OrganisationSchedule.GetScheduleByOrganisationID(organisationID)
	if err != nil {
		return dto.SlotResponse{}, errors.New("organisation schedule has no data for timings")
	}
	slots, err := s.checkIfSlotAvailable(appointments, orgSchedules, date)
	if err != nil {
		return dto.SlotResponse{}, errors.New("processing slot failed, please retry after sometime")
	}
	var slotResponse dto.SlotResponse
	slotResponse.Apptmnt = s.toSlotResponse(slots)
	slotResponse.Message = SlotFetch
	slotResponse.Code = StatusOk
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

	//var availableSlots []Slot
	for i, slot := range allSlots {
		if s.isSlotOccupied(slot, appointments) {
			allSlots[i].Allow = false
			continue
		}
		//availableSlots = append(availableSlots, slot)
	}

	return allSlots, nil
}

func (s *AppointmentService) createSlots(schedule admindto.GetResponse, date string) ([]Slot, error) {
	slotDuration := time.Duration(schedule.Slotduration) * time.Minute
	var slots []Slot
	slotStart := schedule.Starttime
	if schedule.Endtime.Before(schedule.Starttime) {
		return nil, errors.New("schedule end time must be after start time")
	}
	currentTime := time.Now()
	currentDate := currentTime.Format(time.DateOnly)
	now := s.normalizeTimeOfDay(currentTime)
	if currentDate == date {
		if !now.Before(schedule.Endtime) {
			return []Slot{}, nil
		}
		if now.After(slotStart) {
			slotStart = now
		}
	}

	for {
		slotEnd := slotStart.Add(slotDuration)
		if slotEnd.After(schedule.Endtime) {
			break
		}

		if s.timesOverlap(slotStart, slotEnd, schedule.BreakStarttime, schedule.BreakEndtime) {
			slotStart = schedule.BreakEndtime.Add(defaultBuffer)
			continue
		}

		slots = append(slots, Slot{Start: slotStart, End: slotEnd, Allow: true})
		slotStart = slotEnd.Add(defaultBuffer)
	}

	return slots, nil
}

func (s *AppointmentService) isSlotOccupied(slot Slot, appointments []Appointment) bool {

	for _, eachAppointment := range appointments {
		appointmentStart := s.normalizeTimeOfDay(eachAppointment.StartTime)
		appointmentEnd := s.normalizeTimeOfDay(eachAppointment.EndTime)
		if s.timesOverlap(slot.Start, slot.End, appointmentStart, appointmentEnd) {
			return true
		}
	}
	return false
}

func (s *AppointmentService) normalizeTimeOfDay(value time.Time) time.Time {
	return time.Date(0, 1, 1, value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), time.UTC)
}

func (s *AppointmentService) timesOverlap(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && startB.Before(endA)
}
func (s *AppointmentService) GetAppointmentsByOrgID(reqModel dto.GetDataReq) ([]dto.AppointmentList, int, error) {
	dblimit, dbpageno := s.parsepagination(reqModel.Limit, reqModel.PageNo)
	reqModel.Dblimit = dblimit
	reqModel.Dbpageno = dbpageno
	query, args := s.buildQueryWithFilters(reqModel)
	data, err := s.Repository.FindManyByOrganisationID(query, args...)
	if err != nil {
		return nil, 0, err
	}
	countQuery := "SELECT COUNT(*) FROM appointments WHERE organisation_id = $1"
	total, err := s.Repository.GetTotalAppointmentsByOrgID(countQuery, reqModel.OrganisationID)
	if err != nil {
		return nil, 0, err
	}
	response := s.toAppointmentList(data)

	return response, total, nil
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
	argsPos := 2
	if reqModel.Date != "" {
		switch reqModel.Date {
		case Today:
			baseQuery += `
		AND a.appointment_date >= CURRENT_DATE
		AND a.appointment_date < CURRENT_DATE + INTERVAL '1 day'
	`

		case Tomorrow:
			baseQuery += `
		AND a.appointment_date >= CURRENT_DATE + INTERVAL '1 day'
		AND a.appointment_date < CURRENT_DATE + INTERVAL '2 day'`

		case ThisWeek:
			baseQuery += `
		AND a.appointment_date >= DATE_TRUNC('week', CURRENT_DATE)
		AND a.appointment_date < DATE_TRUNC('week', CURRENT_DATE) + INTERVAL '7 days'
	`

		case ThisMonth:
			baseQuery += `
		AND a.appointment_date >= DATE_TRUNC('month', CURRENT_DATE)
		AND a.appointment_date < DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month'
	`
		}
	}
	if reqModel.DoctorID != "" {
		baseQuery += fmt.Sprintf(" AND a.doctor_id = $%d", argsPos)
		args = append(args, reqModel.DoctorID)
		argsPos++
	}
	if reqModel.Status != "" {
		baseQuery += fmt.Sprintf(" AND a.status = $%d", argsPos)
		args = append(args, reqModel.Status)
		argsPos++
	}
	if reqModel.VisitType != "" {
		baseQuery += fmt.Sprintf(" AND a.visit_type = $%d", argsPos)
		args = append(args, reqModel.VisitType)
		argsPos++
	}
	baseQuery += " ORDER BY a.start_time ASC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argsPos, argsPos+1)
	args = append(args, reqModel.Dblimit, reqModel.Dbpageno)
	return baseQuery, args
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
	for i, each := range data {
		var singleResp dto.AppointmentList
		singleResp.AppointmentID, _ = each["appointment_id"].(string)
		singleResp.AppointmentCode, _ = each["appointment_code"].(string)
		status, _ := each["status"].(string)
		singleResp.PatientName, _ = each["patient_name"].(string)
		singleResp.DoctorName, _ = each["doctor_name"].(string)
		singleResp.MobileNo, _ = each["mobile_number"].(string)
		singleResp.StartTime, _ = each["start_time"].(time.Time)
		singleResp.EndTime, _ = each["end_time"].(time.Time)

		singleResp.AppointmentDate, _ = each["appointment_date"].(time.Time)
		if i == 0 {
			singleResp.Next = true
		}
		singleResp.VisitType, _ = each["visit_type"].(string)
		singleResp.Status = string(s.findStatus(status, singleResp.EndTime, singleResp.AppointmentDate))
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
	//if appointmentdate is equal to currentdate, then check
	currenttime := time.Now()
	todayDate := time.Date(currenttime.Year(), currenttime.Month(), currenttime.Day(), 0, 0, 0, 0, time.UTC)
	todayendtime := time.Date(currenttime.Year(), currenttime.Month(), currenttime.Day(), endtime.Hour(), endtime.Minute(), endtime.Second(), endtime.Nanosecond(), time.UTC)
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
func (s *AppointmentService) GetAppointmentPreview(organisationID string, appointmentID string) (dto.AppointmentDetails, error) {
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
    pr.medicines,
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
	dbResp, err := s.Repository.GetAppointmentsPreview(query, organisationID, appointmentID)
	if err != nil {
		return dto.AppointmentDetails{}, err
	}
	appointmentDetails := s.toAppointmentPreview(dbResp)
	return appointmentDetails, nil

}
func (s *AppointmentService) toAppointmentPreview(data map[string]interface{}) dto.AppointmentDetails {
	var response dto.AppointmentDetails
	response.AppointmentID, _ = data["appointment_id"].(string)
	response.AppointmentCode, _ = data["appointment_code"].(string)
	response.StartTime, _ = data["start_time"].(time.Time)
	response.EndTime, _ = data["end_time"].(time.Time)
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
	response.Status = string(s.findStatus(response.Status, response.EndTime, response.AppointmentDate))
	response.Medicines = len(medicines)
	return response
}

func (s *AppointmentService) GetAppntmentByID(appointmentID string) (Appointment, error) {
	appointments, err := s.Repository.GetAppointmentByID(appointmentID)
	if err != nil {
		return Appointment{}, err
	}
	return appointments, nil
}
func (s *AppointmentService) UpdateStatus(updateReq dto.UpdateStatus) (err error) {
	Status := s.SelectStatus(updateReq.Status)
	err = s.Repository.UpdateStatus(s.Db, Status, updateReq.AppointmentID)
	if err != nil {
		return
	}
	return
}
func (s *AppointmentService) SelectStatus(status string) Status {
	switch status {
	case "completed":
		return StatusCompleted
	case "cancelled":
		return StatusCancelled
	case "scheduled":
		return StatusScheduled
	case "ongoing":
		return StatusOngoing
	default:
		return StatusScheduled
	}

}
func (s *AppointmentService) GetAppointmentByPatientID(reqModel dto.PatientAppntment) (dto.Response, error) {
	dblimit, dbpageno := s.parsepagination(reqModel.Limit, reqModel.Pageno)
	query, args := s.buidPatientAppntmentFilter(reqModel, dblimit, dbpageno)
	appointments, err := s.Repository.GetAppointmentByPatientID(query, args...)
	if err != nil {
		return dto.Response{}, err
	}
	appointmentCount, err := s.Repository.GetAppointmentByPatientIDCount(reqModel.PatientID, reqModel.OrganisationID)
	if err != nil {
		return dto.Response{}, err
	}
	patAppointments := s.toPatientAppntment(appointments)
	var response dto.Response
	response.Data = patAppointments
	response.Total = int(appointmentCount)
	response.Code = "200"
	response.Message = "fetched successfully"
	return response, nil
}
func (s *AppointmentService) toPatientAppntment(appointments []map[string]interface{}) []dto.PatAppointment {
	var patAppointment []dto.PatAppointment
	for _, each := range appointments {
		var eachAppointment dto.PatAppointment
		eachAppointment.AppointmentID, _ = each["appointment_id"].(string)
		eachAppointment.AppointmentCode, _ = each["appointment_code"].(string)
		eachAppointment.AppointmentDate, _ = each["appointment_date"].(time.Time)
		eachAppointment.StartTime, _ = each["start_time"].(time.Time)
		eachAppointment.DepartmentName, _ = each["name"].(string)
		eachAppointment.DoctorName, _ = each["username"].(string)
		eachAppointment.Status, _ = each["status"].(string)
		eachAppointment.VisitType, _ = each["visit_type"].(string)
		endtime, _ := each["end_time"].(time.Time)
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
