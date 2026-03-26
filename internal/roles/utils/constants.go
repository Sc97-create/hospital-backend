package utils

import (
	"hospital-backend/internal/roles"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultRoleAdmin         = "Super Admin"
	DefaultRoleHospitalAdmin = "Hospital Admin"
	DefaultRoleDoctor        = "Doctor"
	DefaultRoleNurse         = "Nurse"
	DefaultRoleReceptionist  = "Receptionist"
	DefaultRoleLabTechnician = "Lab Technician"
	DefaultRolePharmacist    = "Pharmacist"
)

type RoleArr []string

var DefaultRoleArr RoleArr = RoleArr{
	DefaultRoleAdmin,
	DefaultRoleHospitalAdmin,
	DefaultRoleDoctor,
	DefaultRoleNurse,
	DefaultRoleReceptionist,
	DefaultRoleLabTechnician,
	DefaultRolePharmacist,
}

func AddDefaultRoles(userID string) []roles.Role {
	var defaultRoles []roles.Role
	for _, each := range DefaultRoleArr {
		defaultRoles = append(defaultRoles, roles.Role{
			ID:           uuid.NewString(),
			Name:         each,
			CreatedBy:    userID,
			IsSystemRole: true,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			UpdatedBy:    userID,
			IsDeleted:    false,
			IsDefault:    true,
		})
	}

	return defaultRoles
}
