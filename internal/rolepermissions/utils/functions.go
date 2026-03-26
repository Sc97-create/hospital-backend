package utils

import (
	rolepermission "hospital-backend/internal/rolepermissions"
	"time"

	"github.com/google/uuid"
)

func AddDefaultRolePermissions(roleID string, permArr []string) []rolepermission.RolePermission {
	rolePermissionArr := []rolepermission.RolePermission{}
	for _, perm := range permArr {
		rolePermissionArr = append(rolePermissionArr, rolepermission.RolePermission{
			ID:           uuid.New().String(),
			RoleID:       roleID,
			PermissionID: perm,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
	}
	return rolePermissionArr
}
