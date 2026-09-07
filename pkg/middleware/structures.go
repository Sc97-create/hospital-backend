package middleware

import rpdto "hospital-backend/internal/rolepermissions/dto"

const RoleAccessKey = "role_access"

// RoleAccessLocal is the per-request RBAC snapshot stored in Fiber locals.
type RoleAccessLocal struct {
	IsAdmin  bool
	ByModule map[string]rpdto.ModulePermissionFlags
}

// RoleIDFinder resolves a user's role_id for LoadRoleAccess.
type RoleIDFinder interface {
	FindRoleIDByUserID(userID string) (string, error)
}

// RoleAccessLoader loads module permissions for a role (satisfied by RolePermissionService).
type RoleAccessLoader interface {
	FindModulesByRoleID(roleID string) (rpdto.RoleAccess, error)
}
