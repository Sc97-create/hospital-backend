package appointments

import "time"

type Status string
type VisitType string

const (
	StatusScheduled          Status    = "scheduled"
	StatusCompleted          Status    = "completed"
	StatusCancelled          Status    = "cancelled"
	StatusOngoing            Status    = "ongoing"
	StatusUpcoming           Status    = "upcoming"
	StatusMissed             Status    = "missed"
	StatusReschedule         Status    = "reschedule_required"
	VisitTypeNewPatint       VisitType = "new_patient"
	VisitTypeFollowUp        VisitType = "follow_up"
	VisitTypeOPD             VisitType = "opd"
	AppointmentCreated                 = "appointment created successfully"
	SlotFetch                          = "time slots for appointment fetched successfully"
	AppointmentData                    = "appointment data retrieved successfully"
	StatusOk                           = "200"
	Today                              = "today"
	Tomorrow                           = "tomorrow"
	ThisWeek                           = "this_week"
	ThisMonth                          = "this_month"
	PrescriptionID                     = "prescription_id"
	DbStatus                           = "status"
	AppointmentCreatedEvent            = "appointment_created"
	AppointmentCreateSubject           = "Appointment Created"
)

type Appointment struct {
	ID              string    `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	AppointmentCode string    `json:"appointment_code" gorm:"column:appointment_code;type:text;"`
	ScheduleID      string    `json:"schedule_id" gorm:"type:uuid"`
	SeriesID        string    `json:"series_id" gorm:"column:series_id;"`
	PatientID       string    `json:"patient_id" gorm:"column:patient_id;type:uuid"`
	DoctorID        string    `json:"doctor_id" gorm:"column:doctor_id;type:uuid"`
	OrganisationID  string    `json:"organisation_id" gorm:"column:organisation_id;type:uuid"`
	AppointmentDate time.Time `json:"appointment_date" gorm:"type:timestamp;column:appointment_date"`
	StartTime       time.Time `json:"start_time" gorm:"column:start_time;type:timestamptz"`
	EndTime         time.Time `json:"end_time" gorm:"column:end_time;type:timestamptz"`
	VisitType       string    `json:"visit_type" gorm:"column:visit_type;type:text"`
	Status          Status    `json:"status" gorm:"column:status;type:text"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;type:timestamp;autoCreateTime"`
	CreatedBy       string    `json:"created_by" gorm:"type:uuid"`
}

type Slot struct {
	Start time.Time
	End   time.Time
	Allow bool
}
type AppointmentFilters struct {
	Date      string
	Status    string
	VisitType string
	Limit     int
	PageNo    int
}
