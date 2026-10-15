package diagnosticcentre

import "time"

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

type DiagnosticCentre struct {
	ID             string    `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	OrganisationID string    `json:"organisation_id" gorm:"column:organisation_id;type:uuid;not null;uniqueIndex:idx_org_diagnostic_centre_name,priority:1"`
	Name           string    `json:"name" gorm:"column:name;type:varchar(150);not null;uniqueIndex:idx_org_diagnostic_centre_name,priority:2"`
	Address        string    `json:"address" gorm:"column:address;type:text"`
	ContactNumber  string    `json:"contact_number" gorm:"column:contact_number;type:varchar(20)"`
	Email          string    `json:"email" gorm:"column:email;type:varchar(150)"`
	Status         string    `json:"status" gorm:"column:status;type:varchar(50);not null;default:active"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (DiagnosticCentre) TableName() string {
	return "diagnostic_centres"
}
