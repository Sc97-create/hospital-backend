package organisation

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type Organisation struct {
	ID               string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code             string    `json:"code" gorm:"type:text"`
	HospitalType     string    `json:"hospital_type" gorm:"type:text"`
	LegalEntityName  string    `json:"legal_entity_name" gorm:"type:text"`
	OrganisationName string    `json:"organisation_name" gorm:"type:varchar(255);not null"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Address          Location  `json:"address" gorm:"type:jsonb"`
	Security         Security  `json:"security" gorm:"type:jsonb"`
}
type Location struct {
	CountryID string    `json:"country_id" gorm:"type:text"`
	StateID   string    `json:"state" gorm:"type:text"`
	CityID    string    `json:"city" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	CreatedBy string    `json:"created_by" gorm:"type:uuid"`
}
type Security struct {
	EnableAuditLog  bool `json:"enabel_audit_logs"`
	EmergencyAccess bool `json:"emergency_access"`
}

func (l *Location) Scan(value interface{}) error {
	if value == nil {
		*l = Location{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, l)
	case string:
		return json.Unmarshal([]byte(v), l)
	default:
		return errors.New("unsupported type for Location")
	}
}

func (l Location) Value() (driver.Value, error) {
	return json.Marshal(l)
}
func (S *Security) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, S)
	case string:
		return json.Unmarshal([]byte(v), S)
	default:
		return errors.New("unsupported type for security")
	}
}
func (S Security) Value() (driver.Value, error) {
	return json.Marshal(S)
}
