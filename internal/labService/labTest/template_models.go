package labservice

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// LabTemplate is a single catalog/test definition (defaults map + jsonb template items).
type LabTemplate struct {
	TestCode    string   `json:"test_code"`
	TestName    string   `json:"test_name"`
	Specimens   []string `json:"specimens"`
	Type        string   `json:"type"`
	TATMinutes  int      `json:"tat_minutes"`
	Description string   `json:"description,omitempty"`
}

// LabTemplateList is a jsonb array of LabTemplate rows on LabTestTemplate.
type LabTemplateList []LabTemplate

func (l *LabTemplateList) Scan(value interface{}) error {
	if value == nil {
		*l = make(LabTemplateList, 0)
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, l)
	case string:
		return json.Unmarshal([]byte(v), l)
	default:
		return errors.New("unsupported type for LabTemplateList")
	}
}

func (l LabTemplateList) Value() (driver.Value, error) {
	if l == nil {
		return json.Marshal(LabTemplateList{})
	}
	return json.Marshal(l)
}

// LabTestTemplate is an organisation-scoped named bundle of lab tests (jsonb).
type LabTestTemplate struct {
	ID             string          `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name           string          `json:"name" gorm:"column:name;type:varchar(200);not null"`
	Description    string          `json:"description" gorm:"column:description;type:text"`
	Tests          LabTemplateList `json:"tests" gorm:"column:tests;type:jsonb;not null"`
	Status         string          `json:"status" gorm:"column:status;type:varchar(50);not null;default:active"`
	OrganisationID string          `json:"organisation_id" gorm:"column:organisation_id;type:uuid;not null;index"`
	CreatedBy      string          `json:"created_by" gorm:"column:created_by;type:uuid;not null"`
	CreatedAt      time.Time       `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (LabTestTemplate) TableName() string {
	return "lab_test_templates"
}
