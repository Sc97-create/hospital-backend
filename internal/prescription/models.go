package prescription

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type MedicineList []Medicines
type TabletFreq Freq

type Prescription struct {
	ID              string       `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	Code            string       `json:"code" gorm:"column:code"`
	PatientID       string       `json:"patient_id" gorm:"type:uuid;column:patient_id"`
	AppointmentID   string       `json:"appointment_id" gorm:"type:uuid"`
	PrescribedBy    string       `json:"prescribed_by" gorm:"type:uuid;column:prescribed_by"`
	OrganisationID  string       `json:"organisation_id" gorm:"type:uuid;column:organisation_id"`
	Status          Status       `json:"status" gorm:"column:status"`
	Medicines       MedicineList `json:"medicines" gorm:"type:jsonb"`
	CreatedAt       time.Time    `json:"created_at" gorm:"column:created_at"`
	FoodInstruction string       `json:"food_instruction" gorm:"column:food_instruction"`
	UpdatedAt       time.Time    `json:"updated_at" gorm:"column:updated_at"`
}

type Freq struct {
	Morning   float64 `json:"morning" gorm:"column:morning"`
	Afternoon float64 `json:"afternoon" gorm:"column:afternoon"`
	Night     float64 `json:"night" gorm:"column:night"`
}

type Medicines struct {
	MedicineID      string  `json:"medicine_id" gorm:"type:uuid;column:medicine_id"`
	MedicineName    string  `json:"medicine_name" gorm:"-"`
	Frequency       Freq    `json:"frequency" gorm:"column:frequency"`
	Quantity        int     `json:"quantity" gorm:"column:quantity"`
	DurationDay     float64 `json:"duration_day" gorm:"column:duration_day"`
	DurationType    string  `json:"duration_type" gorm:"column:duration_type"`
	TabletForm      string  `json:"tablet_form" gorm:"column:tablet_form"`
	FoodInstruction string  `json:"food_instruction" gorm:"column:food_instruction"`
	MedicineType    string  `json:"medicine_type" gorm:"column:medicine_type"`
	Dosage          string  `json:"dosage" gorm:"column:dosage"`
}

func (S *MedicineList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, S)
	case string:
		return json.Unmarshal([]byte(v), S)
	default:
		return errors.New("unsupported type for medicines")
	}
}
func (S MedicineList) Value() (driver.Value, error) {
	return json.Marshal(S)
}
func (F *TabletFreq) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, F)
	case string:
		return json.Unmarshal([]byte(v), F)
	default:
		return errors.New("unsupported type for freq")
	}
}
func (F *Freq) Value() (driver.Value, error) {
	return json.Marshal(F)
}
