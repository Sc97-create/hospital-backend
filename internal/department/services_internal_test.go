package department

import "testing"

func TestCreateDeptArray(t *testing.T) {
	svc := NewDepartmentService(nil)
	depts := svc.createDeptArray("org-1")

	if len(depts) != len(DefaultDeptArr) {
		t.Fatalf("expected %d departments, got %d", len(DefaultDeptArr), len(depts))
	}

	names := make(map[string]bool)
	for _, d := range depts {
		if d.OrganisationID != "org-1" {
			t.Fatalf("expected org-1, got %q", d.OrganisationID)
		}
		if d.ID == "" {
			t.Fatal("expected generated id")
		}
		if !d.IsActive {
			t.Fatal("expected active department")
		}
		names[d.Name] = true
	}

	for _, name := range DefaultDeptArr {
		if !names[name] {
			t.Fatalf("missing default department %q", name)
		}
	}
}
