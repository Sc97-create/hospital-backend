package roles

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID             string         `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name           string         `json:"name" gorm:"column:name;type:varchar(150);not null;uniqueIndex:idx_role_name"`
	Description    string         `json:"description" gorm:"column:description;type:text"`
	IsSystemRole   bool           `json:"is_system_role" gorm:"column:is_system_role;default:false"`
	IsActive       bool           `json:"is_active" gorm:"column:is_active;default:true"`
	OrganisationID string         `json:"organisation_id" gorm:"column:organisation_id;"`
	CreatedAt      time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	CreatedBy      string         `json:"created_by" gorm:"column:created_by;type:uuid"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	UpdatedBy      string         `json:"updated_by" gorm:"column:updated_by;type:uuid"`
	IsDeleted      bool           `json:"is_deleted" gorm:"column:is_deleted;default:false"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index"`
	IsDefault      bool           `json:"is_default" gorm:"column:is_default;default:false"`
}
