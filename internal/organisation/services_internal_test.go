package organisation

import (
	"testing"

	dto "hospital-backend/internal/organisation/DTO"
)

func TestCreateOrgModel(t *testing.T) {
	svc := NewOrganisationService(nil, nil, nil, nil, nil, nil, nil)
	payload := dto.OrganisationPayload{
		OrganisationName: "City Hospital",
		LegalEntityName:  "City Hospital LLC",
		HospitalType:     "general",
	}

	org := svc.createOrgModel(payload)
	if org.ID == "" || org.Code == "" {
		t.Fatal("expected generated id and code")
	}
	if org.OrganisationName != payload.OrganisationName {
		t.Fatalf("expected name %q, got %q", payload.OrganisationName, org.OrganisationName)
	}
	if org.LegalEntityName != payload.LegalEntityName {
		t.Fatalf("expected legal entity %q, got %q", payload.LegalEntityName, org.LegalEntityName)
	}
	if org.HospitalType != payload.HospitalType {
		t.Fatalf("expected hospital type %q, got %q", payload.HospitalType, org.HospitalType)
	}
	if org.CreatedAt.IsZero() || org.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be set")
	}
}

func TestCreateOrgModelEmptyPayload(t *testing.T) {
	svc := NewOrganisationService(nil, nil, nil, nil, nil, nil, nil)
	org := svc.createOrgModel(dto.OrganisationPayload{})
	if org.ID == "" {
		t.Fatal("expected id even for empty payload")
	}
}
