package organisations

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Organisation is the central-db facility record (one hospital per org).
type Organisation struct {
	ID               string      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID         string      `json:"tenant_id" gorm:"type:uuid;index"`
	IsPrimary        bool        `json:"is_primary" gorm:"column:is_primary;default:false"`
	LegalEntityName  string      `json:"legal_entity_name" gorm:"type:varchar(255)"`
	OrganisationType string      `json:"organisation_type" gorm:"type:text"`
	FacilityName     string      `json:"facility_name" gorm:"type:varchar(255)"`
	RegistrationNo   string      `json:"registration_no" gorm:"type:varchar(100)"`
	LicenseNumber    string      `json:"license_number" gorm:"type:varchar(100)"`
	LicenseExpiry    *time.Time  `json:"license_expiry" gorm:"type:date"`
	GSTIN            string      `json:"gstin" gorm:"type:varchar(20)"`
	Address          Address     `json:"address" gorm:"type:jsonb"`
	DataSharing      DataSharing `json:"data_sharing" gorm:"type:jsonb"`
	Status           string      `json:"status" gorm:"type:text;default:'active'"`
	CreatedAt        time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Organisation) TableName() string {
	return "organisations"
}

type Address struct {
	Address1  string    `json:"address1"`
	Address2  string    `json:"address2"`
	CountryID string    `json:"country_id"`
	State     string    `json:"state"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

type DataSharing struct {
	PatientLookup bool `json:"patient_lookup"`
	LabReports    bool `json:"lab_reports"`
}

func (a *Address) Scan(value interface{}) error {
	if value == nil {
		*a = Address{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return errors.New("unsupported type for Address")
	}
}

func (a Address) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (d *DataSharing) Scan(value interface{}) error {
	if value == nil {
		*d = DataSharing{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return errors.New("unsupported type for DataSharing")
	}
}

func (d DataSharing) Value() (driver.Value, error) {
	return json.Marshal(d)
}
