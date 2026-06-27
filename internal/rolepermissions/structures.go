package rolepermissions

import "time"

type RolePermission struct {
	ID             string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	RoleID         string    `gorm:"type:uuid;not null" json:"role_id"`
	PermissionID   string    `gorm:"type:uuid;not null" json:"permission_id"`
	ModuleID       string    `gorm:"type:uuid;not null" json:"module_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      time.Time `gorm:"-" json:"-"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
}
type RolePermissionSeed struct {
	Module      string
	Permissions []string
}
