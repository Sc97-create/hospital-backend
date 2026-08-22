package billing

import (
	"errors"
	"testing"

	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/prescription"
	prescriptiondto "hospital-backend/internal/prescription/dto"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type stubInvoiceItemRepo struct {
	createErr error
	lastItems []InvoiceItem
}

func (s *stubInvoiceItemRepo) Create(_ *zap.Logger, _ *gorm.DB, items []InvoiceItem) error {
	s.lastItems = items
	return s.createErr
}

func (s *stubInvoiceItemRepo) GetInvoiceItemsByInvoiceID(_ *zap.Logger, _ string, _ ...interface{}) ([]dto.MedInvoiceItemResponse, error) {
	return nil, nil
}

type stubPrescriptionQty struct {
	qtyMap map[string]prescriptiondto.PrescriptionQtyInfo
	err    error
}

func (s stubPrescriptionQty) UpdatePrescriptionItemByID(_ *zap.Logger, _ prescriptiondto.UpdatePrescriptionItemRequest) error {
	return nil
}

func (s stubPrescriptionQty) GetPrescriptionsByPIDWithLimit(_ *zap.Logger, _ string, _, _ float64) ([]prescription.MixedPrescriptionItem, int64, error) {
	return nil, 0, nil
}

func (s stubPrescriptionQty) GetMedicineInfo(_ *zap.Logger, _ string) ([]prescription.MedicineDetInfo, int64, error) {
	return nil, 0, nil
}

func (s stubPrescriptionQty) GetqtyByMedicine(_ *zap.Logger, _ string) (map[string]prescriptiondto.PrescriptionQtyInfo, error) {
	return s.qtyMap, s.err
}

func TestToInvoiceItem(t *testing.T) {
	svc := &InvoiceItemServ{}
	items := []dto.DispensedItem{servicetestDispensedItem()}

	got := svc.toInvoiceItem("inv-1", items)
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].InvoiceID != "inv-1" || got[0].MedicineID != "med-1" {
		t.Fatalf("unexpected mapping: %+v", got[0])
	}
	if got[0].DispensedQty != 2 {
		t.Fatalf("expected dispensed qty 2, got %d", got[0].DispensedQty)
	}
}

func servicetestDispensedItem() dto.DispensedItem {
	return dto.DispensedItem{
		MedicineID:          "med-1",
		MedicineInventoryID: "inv-1",
		PrescriptionItemID:  "pi-1",
		BatchNo:             "B1",
		QuantitySoldUnits:   2,
		CurrentStockUnits:   10,
		ComputedItemTotal:   100,
		TotalAmount:         110,
	}
}

func TestAddInvoiceItems(t *testing.T) {
	tests := []struct {
		name    string
		items   []dto.DispensedItem
		qtyMap  map[string]prescriptiondto.PrescriptionQtyInfo
		qtyErr  error
		repoErr error
		wantErr error
	}{
		{
			name:   "prescription qty lookup error",
			qtyErr: errors.New("lookup fail"),
			items:  []dto.DispensedItem{servicetestDispensedItem()},
		},
		{
			name:    "medicine not in prescription",
			qtyMap:  map[string]prescriptiondto.PrescriptionQtyInfo{},
			items:   []dto.DispensedItem{servicetestDispensedItem()},
			wantErr: wrapError.ErrMedicineNotInPrescription,
		},
		{
			name: "qty exceeds remaining",
			qtyMap: map[string]prescriptiondto.PrescriptionQtyInfo{
				"med-1": {BalanceAfterDispense: 1},
			},
			items:   []dto.DispensedItem{servicetestDispensedItem()},
			wantErr: wrapError.ErrQtyExceedsRemaining,
		},
		{
			name: "insufficient stock",
			qtyMap: map[string]prescriptiondto.PrescriptionQtyInfo{
				"med-1": {BalanceAfterDispense: 5},
			},
			items: []dto.DispensedItem{{
				MedicineID: "med-1", QuantitySoldUnits: 2, CurrentStockUnits: 1,
			}},
			wantErr: wrapError.ErrInsufficientStock,
		},
		{
			name: "zero qty still persisted",
			qtyMap: map[string]prescriptiondto.PrescriptionQtyInfo{
				"med-1": {BalanceAfterDispense: 5},
			},
			items: []dto.DispensedItem{{
				MedicineID: "med-1", QuantitySoldUnits: 0, CurrentStockUnits: 10,
			}},
		},
		{
			name: "repo create error",
			qtyMap: map[string]prescriptiondto.PrescriptionQtyInfo{
				"med-1": {BalanceAfterDispense: 5},
			},
			items:   []dto.DispensedItem{servicetestDispensedItem()},
			repoErr: errors.New("db"),
		},
		{
			name: "success",
			qtyMap: map[string]prescriptiondto.PrescriptionQtyInfo{
				"med-1": {BalanceAfterDispense: 5},
			},
			items: []dto.DispensedItem{servicetestDispensedItem()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubInvoiceItemRepo{createErr: tt.repoErr}
			svc := &InvoiceItemServ{
				InvItemRepo:      repo,
				PrescriptionItem: stubPrescriptionQty{qtyMap: tt.qtyMap, err: tt.qtyErr},
			}
			err := svc.addInvoiceItems(nil, nil, "rx-1", "inv-1", tt.items)
			if tt.qtyErr != nil {
				if err == nil {
					t.Fatal("expected qty lookup error")
				}
				return
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if tt.repoErr != nil {
				if err == nil {
					t.Fatal("expected repo error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.name == "zero qty still persisted" && len(repo.lastItems) != 1 {
				t.Fatalf("expected zero-qty item persisted, got %d", len(repo.lastItems))
			}
			if tt.name == "success" && len(repo.lastItems) != 1 {
				t.Fatalf("expected 1 persisted item, got %d", len(repo.lastItems))
			}
		})
	}
}
