package rolepermissions

import (
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/constants"
)

// moduleActions is the set of CRUD actions a role gets on one module.
type moduleActions struct {
	Module  string
	Actions []string
}

// defaultRolePermissionMatrix is the single place to grant module actions to default roles.
var defaultRolePermissionMatrix = map[string][]moduleActions{
	roles.DefaultRoleDoctor: {
		{Module: constants.Appointment, Actions: []string{permissions.View, permissions.Update}},
		{Module: constants.Prescription, Actions: []string{permissions.Create, permissions.Update, permissions.View}},
		{Module: constants.Patient, Actions: []string{permissions.View, permissions.Update}},
		{Module: constants.Employee, Actions: []string{permissions.View}},
		{Module: constants.Medicine, Actions: []string{permissions.View}},
		{Module: constants.Dashboard, Actions: []string{permissions.View}},
	},
	roles.DefaultRolePharmacist: {
		{Module: constants.Medicine, Actions: []string{permissions.View, permissions.Update, permissions.Create}},
		{Module: constants.Prescription, Actions: []string{permissions.View, permissions.Update}},
		{Module: constants.Billing, Actions: []string{permissions.View, permissions.Update, permissions.Create}},
	},
	roles.DefaultRoleReceptionist: {
		{Module: constants.Patient, Actions: []string{permissions.Create, permissions.View, permissions.Update}},
		{Module: constants.Appointment, Actions: []string{permissions.Create, permissions.View, permissions.Update}},
		// view+create = checkout (POST /billing/create); update = payment confirm / retry link
		{Module: constants.Billing, Actions: []string{permissions.View, permissions.Create, permissions.Update}},
		{Module: constants.Dashboard, Actions: []string{permissions.View}},
	},
	roles.DefaultRoleNurse: {
		{Module: constants.Patient, Actions: []string{permissions.View}},
		{Module: constants.Appointment, Actions: []string{permissions.View}},
	},
	// DefaultRoleHospitalAdmin / DefaultRoleLabTechnician: no default module grants yet.
	// DefaultRoleAdmin is handled separately via is_admin (full access).
}
