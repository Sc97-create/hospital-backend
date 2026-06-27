package prescription

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type MedicineList []PrescriptionItems
type TabletFreq Freq
type MedBatchList []MedicineBatch

type Prescription struct {
	ID             string `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	Code           string `json:"code" gorm:"column:code"`
	PatientID      string `json:"patient_id" gorm:"type:uuid;column:patient_id"`
	AppointmentID  string `json:"appointment_id" gorm:"type:uuid"`
	PrescribedBy   string `json:"prescribed_by" gorm:"type:uuid;column:prescribed_by"`
	OrganisationID string `json:"organisation_id" gorm:"type:uuid;column:organisation_id"`
	//Medicines       MedicineList `json:"medicines" gorm:"type:jsonb"`
	Status    Status    `json:"status" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`

	UpdatedAt time.Time           `json:"updated_at" gorm:"column:updated_at"`
	Items     []PrescriptionItems `gorm:"foreignKey:PrescriptionID"`
}

type Freq struct {
	Morning   float64 `json:"morning" gorm:"column:morning"`
	Afternoon float64 `json:"afternoon" gorm:"column:afternoon"`
	Night     float64 `json:"night" gorm:"column:night"`
}

type PrescriptionItems struct {
	ID                   string    `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	PrescriptionID       string    `json:"prescription_id" gorm:"type:uuid;"`
	MedicineID           string    `json:"medicine_id" gorm:"type:uuid"`
	Frequency            Freq      `json:"frequency" gorm:"column:frequency;type:jsonb"`
	Quantity             int       `json:"quantity" gorm:"column:quantity"`
	DurationDay          float64   `json:"duration_day" gorm:"column:duration_day"`
	DurationType         string    `json:"duration_type" gorm:"column:duration_type"`
	FoodInstruction      string    `json:"food_instruction" gorm:"column:food_instruction"`
	Status               Status    `json:"status" gorm:"column:status"`
	BalanceAfterDispense int       `json:"balance_after_dispense" gorm:"column:balance_after_dispense"`
	CreatedAt            time.Time `json:"created_at" gorm:"type:timestamptz"`
	CreatedBy            string    `json:"created_by" gorm:"type:uuid"`

	Prescription Prescription `gorm:"foreignKey:PrescriptionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
type MixPrescriptionData struct {
	DoctorName     string       `json:"doctor_name"`
	ID             string       `json:"id"`
	Reason         string       `json:"reason"`
	CreatedAt      time.Time    `json:"created_at"`
	Medicines      MedicineList `json:"medicines"`
	PatientID      string       `json:"patient_id"`
	OrganisationID string       `json:"organisation_id"`
}
type MixedPrescriptionItem struct {
	PrescriptionID  string  `json:"prescription_id"`
	Frequency       Freq    `json:"frequency"`
	DurationDay     float64 `json:"duration_day"`
	DurationType    string  `json:"duration_type"`
	Quantity        int     `json:"quantity"`
	FoodInstruction string  `json:"food_instruction"`
	MedicineName    string  `json:"medicine_name"`
	MedicineForm    string  `json:"medicine_form"`
	MedicineID      string  `json:"medicine_id"`
}
type MedicineDetInfo struct {
	PrescriptionCode      string       `json:"prescription_code"`
	PrescriptionStatus    string       `json:"prescription_status"`
	PrescriptionCreatedAt time.Time    `json:"prescription_created_at"`
	PrescribedQuantity    int          `json:"prescribed_quantity"`
	PrescriptionID        string       `json:"prescription_id"`
	PrescriptionItemID    string       `json:"prescription_item_id"`
	MedicineID            string       `json:"medicine_id"`
	MedicineName          string       `json:"medicine_name"`
	MedicineForm          string       `json:"medicine_form"`
	MedicineStrength      string       `json:"medicine_strength"`
	Frequency             Freq         `json:"frequency" gorm:"type:jsonb"`
	ReorderLevel          int          `json:"reorder_level"`
	MaxStockTarget        int          `json:"max_stock_target"`
	MedicineBatches       MedBatchList `json:"medicine_batches" gorm:"type:jsonb"`
}
type MedicineBatch struct {
	BatchID           string            `json:"batch_id"`
	BatchNo           string            `json:"batch_no"`
	ExpiresAt         string            `json:"expires_at"`
	CurrentStockUnits int               `json:"current_stock_units"`
	UnitsPerBox       int               `json:"units_per_box"`
	Pricing           DBMedicinePricing `json:"pricing"` // Kept as raw JSONB pricing
	ShelfLocation     string            `json:"shelf_location"`
}
type DBMedicinePricing struct {
	MRP              float64 `json:"mrp"`
	UnitPrice        float64 `json:"unit_price"`    // MRP / units_per_box
	SellingPrice     float64 `json:"selling_price"` // Target strip/box selling price
	UnitSellingPrice float64 `json:"unit_selling_price"`
}

func (al *MedBatchList) Scan(value interface{}) error {
	if value == nil {
		*al = make(MedBatchList, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for ActiveBatchList")
	}
	return json.Unmarshal(bytes, al)
}

func (F *Freq) Scan(value interface{}) error {
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
