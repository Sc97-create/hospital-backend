package middleware

import (
	"context"

	rpdto "hospital-backend/internal/rolepermissions/dto"

	"go.uber.org/zap"
)

const RoleAccessKey = "role_access"

// RoleAccessLocal is the per-request RBAC snapshot stored in Fiber locals.
type RoleAccessLocal struct {
	IsAdmin  bool
	ByModule map[string]rpdto.ModulePermissionFlags
}

// RoleIDFinder resolves a user's role_id for LoadRoleAccess.
type RoleIDFinder interface {
	FindRoleIDByUserID(log *zap.Logger, userID string) (string, error)
}

// RoleAccessLoader loads module permissions for a role (satisfied by RolePermissionService).
type RoleAccessLoader interface {
	FindModulesByRoleID(log *zap.Logger, roleID string) (rpdto.RoleAccess, error)
}

// SubscriptionAccess asks central whether the caller's subscription has ended.
// A true result means hospital usage must be blocked. The message is shown to the user.
type SubscriptionAccess interface {
	Allow(ctx context.Context, userID string) (ended bool, message string, err error)
}
