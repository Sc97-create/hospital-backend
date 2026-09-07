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
	TempPassword string `json:"temp_password" gorm:"column:temp_password;type:text"`

	EmailID string `json:"email_id" gorm:"column:email_id;type:varchar(150);not null;uniqueIndex:idx_org_email"`

	PhoneNumber string `json:"phone_number" gorm:"column:phone_number;type:text"`

	EmployeeCode string `json:"employee_code" gorm:"column:employee_code;type:varchar(50)"`

	Address string `json:"address" gorm:"column:address;type:text"`

	DateOfBirth string `json:"date_of_birth" gorm:"column:date_of_birth;type:varchar(20)"`

	DateOfJoining string `json:"date_of_joining" gorm:"column:date_of_joining;type:varchar(20)"`

	ShiftStartTime string `json:"shift_start_time" gorm:"column:shift_start_time;type:varchar(10)"`
	ShiftEndTime   string `json:"shift_end_time" gorm:"column:shift_end_time;type:varchar(10)"`

	LicenseNo string `json:"license_no" gorm:"column:license_no;type:varchar(50)"`

	Qualification string `json:"qualification" gorm:"column:qualification;type:text"`

	EmployeeType string `json:"employee_type" gorm:"column:employee_type;type:varchar(50)"`

	EmergencyEmail   string `json:"emergency_email" gorm:"column:emergency_email;type:varchar(150)"`
	EmergencyName    string `json:"emergency_name" gorm:"column:emergency_name;type:varchar(150)"`
	EmergencyContact string `json:"emergency_contact" gorm:"column:emergency_contact;type:varchar(50)"`

	OrganisationID string `json:"organisation_id" gorm:"column:organisation_id;type:uuid;not null;index;uniqueIndex:idx_org_username;uniqueIndex:idx_org_email"`

	DepartmentID string `json:"department_id" gorm:"column:department_id;type:uuid;not null;index"`

	RoleID string `json:"role_id" gorm:"column:role_id;type:uuid;not null;index"`

	IsActive bool `json:"is_active" gorm:"column:is_active;default:true"`

	LastLoginAttempt int `json:"last_login_attempt" gorm:"column:last_login_attempt;type:int;default:0"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	Organisation organisation.Organisation `gorm:"foreignKey:OrganisationID;references:ID"`
}

type EmployeeListRow struct {
	ID             string `gorm:"column:id"`
	EmployeeCode   string `gorm:"column:employee_code"`
	Username       string `gorm:"column:username"`
	FirstName      string `gorm:"column:first_name"`
	LastName       string `gorm:"column:last_name"`
	EmailID        string `gorm:"column:email_id"`
	PhoneNumber    string `gorm:"column:phone_number"`
	OrganisationID string `gorm:"column:organisation_id"`
	RoleID         string `gorm:"column:role_id"`
	RoleName       string `json:"role_name" gorm:"column:role_name"`
	DepartmentID   string `gorm:"column:department_id"`
	DepartmentName string `json:"department_name" gorm:"column:department_name"`
	IsActive       bool   `gorm:"column:is_active"`
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
