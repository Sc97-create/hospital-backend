package middleware

import (
	"hospital-backend/internal/permissions"
	"hospital-backend/pkg/constants"
	"testing"
)

func TestResolveRoutePermission_IntentNotMethod(t *testing.T) {
	module, action, ok := ResolveRoutePermission("/api/v1/patients/getPatients")
	if !ok || module != constants.Patient || action != permissions.View {
		t.Fatalf("getPatients: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/employee/addEmployee")
	if !ok || module != constants.Employee || action != permissions.Create {
		t.Fatalf("employee addEmployee: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/prescription/get")
	if !ok || module != constants.Prescription || action != permissions.View {
		t.Fatalf("prescription get POST-style: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/employee/delete")
	if !ok || module != constants.Employee || action != permissions.Delete {
		t.Fatalf("employee delete: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/patients/getpatientByID/abc-123")
	if !ok || module != constants.Patient || action != permissions.View {
		t.Fatalf("param path: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/dashboard/getByStatus")
	if !ok || module != constants.Dashboard || action != permissions.View {
		t.Fatalf("dashboard getByStatus: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/dashboard/getTodayAppointments")
	if !ok || module != constants.Dashboard || action != permissions.View {
		t.Fatalf("dashboard getTodayAppointments: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/dashboard/getTodayInvoiceSummary")
	if !ok || module != constants.Dashboard || action != permissions.View {
		t.Fatalf("dashboard getTodayInvoiceSummary: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/dashboard/getEmployeeStatusCounts")
	if !ok || module != constants.Dashboard || action != permissions.View {
		t.Fatalf("dashboard getEmployeeStatusCounts: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/dashboard/getTodayPrescriptions")
	if !ok || module != constants.Dashboard || action != permissions.View {
		t.Fatalf("dashboard getTodayPrescriptions: got module=%q action=%q ok=%v", module, action, ok)
	}

	_, _, ok = ResolveRoutePermission("/api/v1/bed/createBed")
	if ok {
		t.Fatal("expected bed route to be unmapped")
	}
}

func TestIsPublicRoute(t *testing.T) {
	if !IsPublicRoute("/api/v1/authentication/login") {
		t.Fatal("login should be public")
	}
	if !IsPublicRoute("/api/v1/authentication/requestPasswordReset") {
		t.Fatal("requestPasswordReset should be public")
	}
	if !IsPublicRoute("/api/v1/authentication/updatePassword") {
		t.Fatal("updatePassword should be public to RBAC")
	}
	if !IsPublicRoute("/api/v1/authentication/updatePasswordFirstLogin") {
		t.Fatal("updatePasswordFirstLogin should be public to RBAC")
	}
	if !IsPublicRoute("/api/v1/employee/getDoctors") {
		t.Fatal("getDoctors should be public to RBAC")
	}
	if !IsPublicRoute("/api/v1/employee/findbyID") {
		t.Fatal("findbyID should be public to RBAC")
	}
	if !IsPublicRoute("/api/v1/payment/webhook") {
		t.Fatal("payment webhook should be public")
	}
	if !IsPublicRoute("/api/v1/employee/create") {
		t.Fatal("createAdmin should be public to RBAC")
	}
	if IsPublicRoute("/api/v1/dashboard/getByStatus") {
		t.Fatal("dashboard getByStatus should not be public")
	}
	if IsPublicRoute("/api/v1/employee/addEmployee") {
		t.Fatal("addEmployee should not be public")
	}
}
