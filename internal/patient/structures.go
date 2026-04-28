package patient

import (
	"hospital-backend/internal/organisation"
	"time"

	"github.com/lib/pq"
)

type Status string

const (
	StatusAdmitted   Status = "admitted"
	StatusDischarged Status = "discharged"
)

type Rooms struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomType   string    `json:"room_type" gorm:"not null"`
	WardNumber string    `json:"ward_number" gorm:"not null"`
	Status     string    `json:"status" gorm:"default:'available'"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
type Bed struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomID      string    `json:"room_id" gorm:"type:uuid;not null"`
	Status      string    `json:"status" gorm:"default:'available'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	PricePerDay float64   `json:"price_per_day"`
}

type Patient struct {
	ID              string                    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PatientCode     string                    `json:"patient_code" gorm:"not null"`
	FirstName       string                    `json:"first_name" gorm:"not null"`
	LastName        string                    `json:"last_name"`
	Age             int                       `json:"age" gorm:"not null"`
	Gender          string                    `json:"gender" gorm:"not null"`
	Weight          int                       `json:"weight" gorm:"not null"`
	AdmissionDate   time.Time                 `json:"admission_date" gorm:"not null"`
	DischargeDate   *time.Time                `json:"discharge_date"`
	EmailID         string                    `json:"email_id" gorm:"unique"`
	MobileNumber    string                    `json:"mobile_no" gorm:"unique"`
	Symptoms        pq.StringArray            `json:"symptoms" gorm:"type:text[]"`
	ActiveCondition string                    `json:"active_condition" `
	CreatedBy       string                    `json:"created_by"`
	OrganisationID  string                    `json:"organisation_id"`
	Organisation    organisation.Organisation `gorm:"foreignKey:OrganisationID"`
	DoctorID        string                    `json:"doctor_id"`
	Status          Status                    `json:"status" gorm:"default:'free'"`
}
