package labservice

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"

	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"

	TestStatusPending   = "pending"
	TestStatusCompleted = "completed"
	TestStatusCancelled = "cancelled"
)

// LabTest is the organisation-scoped catalog entry for a lab investigation.
// Kept for schema compatibility; public APIs use LabOrders.
type LabTest struct {
	ID             string    `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	OrganisationID string    `json:"organisation_id" gorm:"column:organisation_id;type:uuid;not null;uniqueIndex:idx_org_lab_test_name_sub,priority:1"`
	Name           string    `json:"name" gorm:"column:name;type:varchar(150);not null;uniqueIndex:idx_org_lab_test_name_sub,priority:2"`
	SubTest        string    `json:"sub_test" gorm:"column:sub_test;type:varchar(150);not null;default:'';uniqueIndex:idx_org_lab_test_name_sub,priority:3"`
	Status         string    `json:"status" gorm:"column:status;type:varchar(50);not null;default:active"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (LabTest) TableName() string {
	return "lab_tests"
}

// LabOrders is a patient lab order; ordered tests are stored as jsonb snapshots.
type LabOrders struct {
	ID                 string             `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	OrganisationID     string             `json:"organisation_id" gorm:"column:organisation_id;type:uuid;not null;index"`
	PatientID          string             `json:"patient_id" gorm:"column:patient_id;type:uuid;not null;index"`
	DoctorID           string             `json:"doctor_id" gorm:"column:doctor_id;type:uuid;not null;index"`
	DiagnosticCentreID string             `json:"diagnostic_centre_id" gorm:"column:diagnostic_centre_id;type:uuid;not null;index"`
	AppointmentID      *string            `json:"appointment_id" gorm:"column:appointment_id;type:uuid;index"`
	LabTests           OrderedLabTestList `json:"lab_tests" gorm:"column:lab_tests;type:jsonb"`
	Status             string             `json:"status" gorm:"column:status;type:varchar(50);not null;default:pending"`
	IsPayment          bool               `json:"is_payment" gorm:"column:is_payment;type:boolean;not null;default:false"`
	CreatedAt          time.Time          `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt          time.Time          `json:"updated_at" gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (LabOrders) TableName() string {
	return "lab_orders"
}

// LabOrderListRow is a joined read model for list/get APIs (includes patient/doctor details).
type LabOrderListRow struct {
	ID                 string             `gorm:"column:id"`
	OrganisationID     string             `gorm:"column:organisation_id"`
	PatientID          string             `gorm:"column:patient_id"`
	PatientName        string             `gorm:"column:patient_name"`
	PatientCode        string             `gorm:"column:patient_code"`
	PatientAge         int                `gorm:"column:patient_age"`
	DoctorID           string             `gorm:"column:doctor_id"`
	DoctorName         string             `gorm:"column:doctor_name"`
	DiagnosticCentreID string             `gorm:"column:diagnostic_centre_id"`
	AppointmentID      *string            `gorm:"column:appointment_id"`
	LabTests           OrderedLabTestList `gorm:"column:lab_tests"`
	Status             string             `gorm:"column:status"`
	IsPayment          bool               `gorm:"column:is_payment"`
	CreatedAt          time.Time          `gorm:"column:created_at"`
	UpdatedAt          time.Time          `gorm:"column:updated_at"`
}

// OrderedLabTest is a snapshot of a lab test embedded on a lab order.
type OrderedLabTest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	SubTest string `json:"sub_test"`
	Status  string `json:"status"`
}

type OrderedLabTestList []OrderedLabTest

func (l *OrderedLabTestList) Scan(value interface{}) error {
	if value == nil {
		*l = make(OrderedLabTestList, 0)
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, l)
	case string:
		return json.Unmarshal([]byte(v), l)
	default:
		return errors.New("unsupported type for OrderedLabTestList")
	}
}

func (l OrderedLabTestList) Value() (driver.Value, error) {
	if l == nil {
		return json.Marshal(OrderedLabTestList{})
	}
	return json.Marshal(l)
}
