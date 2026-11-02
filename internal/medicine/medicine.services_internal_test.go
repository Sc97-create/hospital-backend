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

func TestDiscountAppliesToSellingPrice(t *testing.T) {
	svc := NewMedicineService(nil, nil, nil, nil, nil, nil)
	item := meddto.MedicineInfo{
		MedInventoryID:   "inv-1",
		MedicineID:       "med-1",
		MRP:              100,
		PurchasePrice:    60,
		SellingPrice:     90,
		Discount:         9,
		PurchaseQtyBoxes: 1,
		UnitPerBoxes:     10,
	}

	inventory := svc.toMedicineInventory([]meddto.MedicineInfo{item}, "user-1", "org-1", "sup-1", "pe-1")
	pricing := inventory[0].Pricing
	if pricing.PurchasePrice != 60 || pricing.SellingPrice != 90 {
		t.Fatalf("prices changed: purchase=%v selling=%v", pricing.PurchasePrice, pricing.SellingPrice)
	}
	if pricing.Discount != 9 || pricing.DiscountType != discountTypePercent {
		t.Fatalf("discount=%v type=%s", pricing.Discount, pricing.DiscountType)
	}
	if pricing.TotalPrice != 81.9 {
		t.Fatalf("total price=%v, want 81.9", pricing.TotalPrice)
	}
	if pricing.UnitPrice != 10 {
		t.Fatalf("unit price=%v, want 10", pricing.UnitPrice)
	}

	mvmt := svc.toMedicineMvmt([]meddto.MedicineInfo{item}, "user-1", "org-1")
	if mvmt[0].UnitPriceAtTimeOfMvmt != 8.19 {
		t.Fatalf("movement price=%v, want 8.19", mvmt[0].UnitPriceAtTimeOfMvmt)
	}
	if err := validateDiscounts([]meddto.MedicineInfo{{Discount: 101}}); err == nil {
		t.Fatal("expected discount above 100 to be rejected")
	}
}
