package license

import (
	"time"
)

type License struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null;index"`
	LicenseKey     string    `json:"license_key" gorm:"uniqueIndex;not null"`
	IssuedAt       time.Time `json:"issued_at" gorm:"autoCreateTime"`
	ExpiresAt      time.Time `json:"expires_at" gorm:"autoCreateTime"`
}
