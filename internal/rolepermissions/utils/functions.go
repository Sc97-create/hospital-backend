package utils

import (
	rolepermission "hospital-backend/internal/rolepermissions"
)

//roles
//permissions

// superadmin => all permissions => all modules
// doctor => all permisiions=> patient, appointments => view
// attendant=> view, edit, delete=> appointment, view, edit,delete=> bedmanagement
// finance => view,edit,delete=> billing
// nurse=> view, edit=> patient
// pharmacist=> inventory, prescription=> view, delete, edit

// get all roles
// get all permissions
// for default we are doing this
func AddDefaultRolePermissions() []rolepermission.RolePermission {
	rolePermissionArr := []rolepermission.RolePermission{}

	// for _, perm := range permArr {
	// 	rolePermissionArr = append(rolePermissionArr, rolepermission.RolePermission{
	// 		ID:           uuid.New().String(),
	// 		RoleID:       roleID,
	// 		PermissionID: perm,
	// 		CreatedAt:    time.Now(),
	// 		UpdatedAt:    time.Now(),
	// 	})
	// }
	return rolePermissionArr
}
