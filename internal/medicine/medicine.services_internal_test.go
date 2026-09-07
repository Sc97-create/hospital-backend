package medicine

import (
	"testing"

	meddto "hospital-backend/internal/medicine/dto"
)

func TestToMedicineAddFlagFilters(t *testing.T) {
	svc := NewMedicineService(nil, nil, nil, nil, nil, nil)
	meds := []meddto.MedicineInfo{
		{
			MedicineID: "med-new",
			Name:       "Paracetamol",
			Form:       "tablet",
			Strength:   "500mg",
			Add:        true,
		},
		{
			MedicineID: "med-existing",
			Name:       "Ibuprofen",
			Form:       "tablet",
			Strength:   "400mg",
			Add:        false,
		},
	}

	got := svc.toMedicine(meds, "user-1", "org-1")
	if len(got) != 1 {
		t.Fatalf("expected 1 medicine with Add=true, got %d", len(got))
	}
	if got[0].ID != "med-new" {
		t.Fatalf("expected med-new, got %q", got[0].ID)
	}
	if got[0].OrganisationID != "org-1" || got[0].CreatedBy != "user-1" {
		t.Fatalf("unexpected mapping: %+v", got[0])
	}
	if got[0].Code == "" {
		t.Fatal("expected generated medicine code")
	}
}

func TestToMedicineResponse(t *testing.T) {
	svc := NewMedicineService(nil, nil, nil, nil, nil, nil)
	meds := []Medicine{
		{ID: "med-1", Name: "Paracetamol"},
		{ID: "med-2", Name: "Ibuprofen"},
	}

	got := svc.toMedicineResponse(meds)
	if len(got) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(got))
	}
	if got[0].ID != "med-1" || got[0].Name != "Paracetamol" {
		t.Fatalf("unexpected first response: %+v", got[0])
	}
	if got[1].ID != "med-2" || got[1].Name != "Ibuprofen" {
		t.Fatalf("unexpected second response: %+v", got[1])
	}
}
