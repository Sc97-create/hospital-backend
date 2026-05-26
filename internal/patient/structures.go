package patient

import (
	"hospital-backend/internal/organisation"
	"time"
)

type Status string
type UHID string

var (
	Code UHID = "CLI"
)

const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
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
	ID             string                    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UHID           string                    `json:"uh_id" gorm:"not null;unique"`
	Name           string                    `json:"name" gorm:"type:text;not null"`
	Age            int                       `json:"age" gorm:"not null"`
	Gender         string                    `json:"gender" gorm:"not null"`
	Weight         int                       `json:"weight" gorm:"not null"`
	EmailID        string                    `json:"email_id" gorm:"unique"`
	MobileNumber   string                    `json:"mobile_no" gorm:"unique"`
	BloodGroup     string                    `json:"blood_group" gorm:"type:varchar(10)"`
	LastVisitDate  time.Time                 `json:"last_visit_date"`
	CreatedBy      string                    `json:"created_by" gorm:"type:uuid"`
	OrganisationID string                    `json:"organisation_id" gorm:"type:uuid"`
	Organisation   organisation.Organisation `gorm:"foreignKey:OrganisationID"`
	Status         Status                    `json:"status" gorm:"default:'free'"`
	CreatedAt      time.Time                 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time                 `json:"updated_at" gorm:"autoUpdateTime"`
	Address        string                    `json:"address" gorm:"type:varchar(500)"`
}
