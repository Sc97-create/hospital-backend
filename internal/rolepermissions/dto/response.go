package dto

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
// permission_names is produced as JSON (json_agg) so GORM/pgx can scan it reliably.
type ModulePermissionRow struct {
	ModuleName      string   `gorm:"column:module_name"`
	PermissionNames []string `gorm:"column:permission_names;serializer:json"`
}
