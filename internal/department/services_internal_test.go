package department

import "testing"

func TestCreateDeptArray(t *testing.T) {
	svc := NewDepartmentService(nil)
	depts := svc.createDeptArray()

	if len(depts) != len(DefaultDeptArr) {
		t.Fatalf("expected %d departments, got %d", len(DefaultDeptArr), len(depts))
	}

	names := make(map[string]bool)
	for _, d := range depts {
		if d.FacilityTypeID != nil {
			t.Fatalf("facility type must stay unset, got %q", *d.FacilityTypeID)
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
