package rolepermissions

import (
	"testing"

	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/constants"

	"github.com/lib/pq"
)

func TestMapModulePermissionRows(t *testing.T) {
	rows := []dto.ModulePermissionRow{
		{ModuleName: "patient", PermissionNames: pq.StringArray{"view", "create", "unknown"}},
		{ModuleName: "billing", PermissionNames: pq.StringArray{"update", "delete"}},
	}

	got := mapModulePermissionRows(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(got))
	}
	if !got[0].Permissions.View || !got[0].Permissions.Create {
		t.Fatalf("expected view+create on patient, got %+v", got[0].Permissions)
	}
	if got[0].Permissions.Update || got[0].Permissions.Delete {
		t.Fatalf("did not expect update/delete on patient, got %+v", got[0].Permissions)
	}
	if !got[1].Permissions.Update || !got[1].Permissions.Delete {
		t.Fatalf("expected update+delete on billing, got %+v", got[1].Permissions)
	}
}

func TestApplyPermissionFlag(t *testing.T) {
	flags := dto.ModulePermissionFlags{}
	applyPermissionFlag(&flags, permissions.Create)
	applyPermissionFlag(&flags, permissions.Update)
	applyPermissionFlag(&flags, permissions.View)
	applyPermissionFlag(&flags, permissions.Delete)
	applyPermissionFlag(&flags, "noop")

	if !flags.Create || !flags.Update || !flags.View || !flags.Delete {
		t.Fatalf("expected all known flags set, got %+v", flags)
	}
}

func TestToRolePermModel(t *testing.T) {
	svc := NewRolePermissionService(nil, nil)

	t.Run("empty non-admin returns zero value", func(t *testing.T) {
		got := svc.toRolePermModel("role-1", "", "", "org-1", false)
		if got.ID != "" || got.RoleID != "" {
			t.Fatalf("expected zero RolePermission, got %+v", got)
		}
	})

	t.Run("success", func(t *testing.T) {
		got := svc.toRolePermModel("role-1", "perm-1", "mod-1", "org-1", false)
		if got.ID == "" || got.RoleID != "role-1" || got.OrganisationID != "org-1" {
			t.Fatalf("unexpected mapping: %+v", got)
		}
		if got.PermissionID == nil || *got.PermissionID != "perm-1" {
			t.Fatalf("expected permission id perm-1, got %+v", got.PermissionID)
		}
		if got.ModuleID == nil || *got.ModuleID != "mod-1" {
			t.Fatalf("expected module id mod-1, got %+v", got.ModuleID)
		}
		if got.IsAdmin {
			t.Fatal("expected non-admin")
		}
	})
}

func TestToAdminRolePermModel(t *testing.T) {
	svc := NewRolePermissionService(nil, nil)
	got := svc.toAdminRolePermModel("role-admin", "org-1")
	if got.ID == "" || got.RoleID != "role-admin" || !got.IsAdmin {
		t.Fatalf("unexpected admin model: %+v", got)
	}
	if got.PermissionID != nil || got.ModuleID != nil {
		t.Fatalf("admin should have nil permission/module ids")
	}
}

func TestNullableUUID(t *testing.T) {
	if nullableUUID("") != nil {
		t.Fatal("expected nil for empty string")
	}
	got := nullableUUID("abc")
	if got == nil || *got != "abc" {
		t.Fatalf("expected pointer to abc, got %v", got)
	}
}

func TestCreateRPModel(t *testing.T) {
	svc := NewRolePermissionService(nil, nil)

	roleArr := []roles.Role{
		{ID: "role-admin", Name: roles.DefaultRoleAdmin},
		{ID: "role-doctor", Name: roles.DefaultRoleDoctor},
		{ID: "role-nurse", Name: roles.DefaultRoleNurse},
	}
	permArr := []permissions.Permission{
		{ID: "p-view", Name: permissions.View},
		{ID: "p-create", Name: permissions.Create},
		{ID: "p-update", Name: permissions.Update},
	}
	modArr := []modules.Modules{
		{ID: "m-patient", Name: constants.Patient},
		{ID: "m-appointment", Name: constants.Appointment},
		{ID: "m-prescription", Name: constants.Prescription},
		{ID: "m-employee", Name: constants.Employee},
		{ID: "m-medicine", Name: constants.Medicine},
	}

	rows := svc.createRPModel(roleArr, permArr, modArr, "org-1")
	if len(rows) == 0 {
		t.Fatal("expected seeded rows")
	}

	var adminCount, doctorCount, nurseCount int
	for _, row := range rows {
		switch row.RoleID {
		case "role-admin":
			adminCount++
			if !row.IsAdmin {
				t.Fatal("admin row must be IsAdmin")
			}
		case "role-doctor":
			doctorCount++
		case "role-nurse":
			nurseCount++
		}
	}
	if adminCount != 1 {
		t.Fatalf("expected 1 admin row, got %d", adminCount)
	}
	if doctorCount == 0 || nurseCount == 0 {
		t.Fatalf("expected doctor and nurse rows, doctor=%d nurse=%d", doctorCount, nurseCount)
	}
}

func TestCreateRPModel_ReceptionistBillingCheckout(t *testing.T) {
	svc := NewRolePermissionService(nil, nil)

	roleArr := []roles.Role{{ID: "role-receptionist", Name: roles.DefaultRoleReceptionist}}
	permArr := []permissions.Permission{
		{ID: "p-view", Name: permissions.View},
		{ID: "p-create", Name: permissions.Create},
		{ID: "p-update", Name: permissions.Update},
	}
	modArr := []modules.Modules{
		{ID: "m-patient", Name: constants.Patient},
		{ID: "m-appointment", Name: constants.Appointment},
		{ID: "m-billing", Name: constants.Billing},
	}

	rows := svc.createRPModel(roleArr, permArr, modArr, "org-1")

	billingPerms := make(map[string]bool)
	for _, row := range rows {
		if row.RoleID != "role-receptionist" || row.ModuleID == nil || *row.ModuleID != "m-billing" {
			continue
		}
		if row.PermissionID != nil {
			billingPerms[*row.PermissionID] = true
		}
	}
	for _, want := range []string{"p-view", "p-create", "p-update"} {
		if !billingPerms[want] {
			t.Fatalf("receptionist missing billing permission %q for checkout", want)
		}
	}
}

func TestCreateRPModel_ReceptionistDashboardView(t *testing.T) {
	svc := NewRolePermissionService(nil, nil)

	roleArr := []roles.Role{{ID: "role-receptionist", Name: roles.DefaultRoleReceptionist}}
	permArr := []permissions.Permission{{ID: "p-view", Name: permissions.View}}
	modArr := []modules.Modules{{ID: "m-dashboard", Name: constants.Dashboard}}

	rows := svc.createRPModel(roleArr, permArr, modArr, "org-1")

	var dashboardView bool
	for _, row := range rows {
		if row.RoleID != "role-receptionist" || row.ModuleID == nil || *row.ModuleID != "m-dashboard" {
			continue
		}
		if row.PermissionID != nil && *row.PermissionID == "p-view" {
			dashboardView = true
		}
	}
	if !dashboardView {
		t.Fatal("receptionist missing dashboard view permission")
	}
}

func TestCreateRPModel_DoctorDashboardView(t *testing.T) {
	svc := NewRolePermissionService(nil, nil)

	roleArr := []roles.Role{{ID: "role-doctor", Name: roles.DefaultRoleDoctor}}
	permArr := []permissions.Permission{{ID: "p-view", Name: permissions.View}}
	modArr := []modules.Modules{
		{ID: "m-dashboard", Name: constants.Dashboard},
		{ID: "m-appointment", Name: constants.Appointment},
	}

	rows := svc.createRPModel(roleArr, permArr, modArr, "org-1")

	var dashboardView bool
	for _, row := range rows {
		if row.RoleID != "role-doctor" || row.ModuleID == nil || *row.ModuleID != "m-dashboard" {
			continue
		}
		if row.PermissionID != nil && *row.PermissionID == "p-view" {
			dashboardView = true
		}
	}
	if !dashboardView {
		t.Fatal("doctor missing dashboard view permission")
	}
}
