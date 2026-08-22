package roles

import (
	"testing"
)

func TestCreateRoleArray(t *testing.T) {
	svc := NewRoleServices(nil)
	roles := svc.createRoleArray("org-1")
	if len(roles) != len(DefaultRoleArr) {
		t.Fatalf("expected %d roles, got %d", len(DefaultRoleArr), len(roles))
	}
	for i, name := range DefaultRoleArr {
		if roles[i].Name != name {
			t.Fatalf("expected role %q, got %q", name, roles[i].Name)
		}
		if roles[i].OrganisationID != "org-1" {
			t.Fatalf("expected org-1, got %q", roles[i].OrganisationID)
		}
		if roles[i].ID == "" {
			t.Fatalf("expected generated id for role %q", name)
		}
	}
}

func TestMapToRoleResponse(t *testing.T) {
	svc := NewRoleServices(nil)
	resp := svc.mapToRoleResponse(Role{ID: "r1", Name: "Doctor"})
	if resp.ID != "r1" || resp.Name != "Doctor" {
		t.Fatalf("unexpected mapping: %+v", resp)
	}
}

func TestArrayMapToRoleResponse(t *testing.T) {
	svc := NewRoleServices(nil)
	list := svc.arrayMapToRoleResponse([]Role{
		{ID: "r1", Name: "Doctor"},
		{ID: "r2", Name: "Nurse"},
	})
	if len(list) != 2 || list[1].Name != "Nurse" {
		t.Fatalf("unexpected list: %+v", list)
	}
}
