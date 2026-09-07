package dto

import "github.com/lib/pq"

type ModulePermissionFlags struct {
	Create bool `json:"create"`
	Update bool `json:"update"`
	View   bool `json:"view"`
	Delete bool `json:"delete"`
}

type RoleModulePermission struct {
	ModuleName  string                `json:"module_name"`
	Permissions ModulePermissionFlags `json:"permissions"`
}

type RoleAccess struct {
	IsAdmin     bool                   `json:"is_admin"`
	Permissions []RoleModulePermission `json:"permissions"`
}

// ModulePermissionRow is the flat scan target for FindModulePermissionsByRoleID raw SQL.
type ModulePermissionRow struct {
	ModuleName      string         `gorm:"column:module_name"`
	PermissionNames pq.StringArray `gorm:"column:permission_names"`
}
