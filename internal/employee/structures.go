package employee

import (
	"hospital-backend/internal/organisation"
	"time"
)

type User struct {
	ID string `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`

	Username string `json:"username" gorm:"column:username;type:varchar(100);not null;uniqueIndex:idx_org_username"`

	FirstName string `json:"first_name" gorm:"column:first_name;type:text"`
	LastName  string `json:"last_name" gorm:"column:last_name;type:text"`

	PasswordHash string `json:"-" gorm:"column:password_hash;type:text"`

	EmailID string `json:"email_id" gorm:"column:email_id;type:varchar(150);not null;uniqueIndex:idx_org_email"`

	PhoneNumber string `json:"phone_number" gorm:"column:phone_number;type:text"`

	OrganisationID string `json:"organisation_id" gorm:"column:organisation_id;type:uuid;not null;index;uniqueIndex:idx_org_username;uniqueIndex:idx_org_email"`

	DepartmentID string `json:"department_id" gorm:"column:department_id;type:uuid;not null;index"`

	RoleID string `json:"role_id" gorm:"column:role_id;type:uuid;not null;index"`

	IsActive bool `json:"is_active" gorm:"column:is_active;default:true"`

	LastLoginAttempt int `json:"last_login_attempt" gorm:"column:last_login_attempt;type:int;default:0"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	Organisation organisation.Organisation `gorm:"foreignKey:OrganisationID;references:ID"`
	//Department   Department                `gorm:"foreignKey:DepartmentID;references:ID"`
	//Role         Role                      `gorm:"foreignKey:RoleID;references:ID"`
}

type InviteEmp struct {
	EmployeeName     string
	OrganisationName string
	Department       string
	Role             string
	EmailID          string
	Password         string
	LoginURL         string
	AppName          string
}
