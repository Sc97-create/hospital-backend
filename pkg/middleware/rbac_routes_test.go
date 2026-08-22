package middleware

import (
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"testing"
)

func TestResolveRoutePermission_IntentNotMethod(t *testing.T) {
	module, action, ok := ResolveRoutePermission("/api/v1/patients/getPatients")
	if !ok || module != modules.Patient || action != permissions.View {
		t.Fatalf("getPatients: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/employee/create")
	if !ok || module != modules.Employee || action != permissions.Create {
		t.Fatalf("employee create: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/prescription/get")
	if !ok || module != modules.Prescription || action != permissions.View {
		t.Fatalf("prescription get POST-style: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/employee/delete")
	if !ok || module != modules.Employee || action != permissions.Delete {
		t.Fatalf("employee delete: got module=%q action=%q ok=%v", module, action, ok)
	}

	module, action, ok = ResolveRoutePermission("/api/v1/patients/getpatientByID/abc-123")
	if !ok || module != modules.Patient || action != permissions.View {
		t.Fatalf("param path: got module=%q action=%q ok=%v", module, action, ok)
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
	if IsPublicRoute("/api/v1/employee/create") {
		t.Fatal("employee create should not be public")
	}
}
