package department

import (
	"time"

	"gorm.io/gorm"
)

type Department struct {
	ID string `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`

	Name string `json:"name" gorm:"column:name;type:varchar(150);not null;"`

	Description string `json:"description" gorm:"column:description;type:text"`

	IsActive bool `json:"is_active" gorm:"column:is_active;default:true"`

	OrganisationID string    `json:"organisation_id" gorm:"column:organisation_id;type:uuid"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	CreatedBy      string    `json:"created_by" gorm:"column:created_by"`

	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	UpdatedBy string    `json:"updated_by" gorm:"column:updated_by"`

	DeletedAt gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index"`
}
