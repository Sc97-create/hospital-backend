package rolepermissions

import "time"

type RolePermission struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	RoleID       string    `gorm:"type:uuid;not null;uniqueIndex:idx_role_permission" json:"role_id"`
	PermissionID string    `gorm:"type:uuid;not null;uniqueIndex:idx_role_permission" json:"permission_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    time.Time `gorm:"-" json:"-"`
}
