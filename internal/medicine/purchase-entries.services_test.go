package medicine_test

import (
	"errors"
	"testing"
	"time"

	"hospital-backend/internal/medicine"
	"hospital-backend/internal/medicine/mocks"
	"hospital-backend/internal/testutil/servicetest"

	"go.uber.org/mock/gomock"
)

func TestCreatePurchaseEntry(t *testing.T) {
	entry := &medicine.MPurchaseEntry{
		ID:             "pe-1",
		InvoiceNumber:  "INV-001",
		InvoiceDate:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		SupplierID:     "sup-1",
		OrganisationID: "org-1",
	}

	tests := []struct {
		name    string
		setup   func(*mocks.MockRPurchaseEntry)
		wantErr bool
	}{
		{
			name: "delegation error",
			setup: func(m *mocks.MockRPurchaseEntry) {
				m.EXPECT().CreatePurchaseEntry(gomock.Any(), gomock.Any(), entry).Return(errors.New("insert failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *mocks.MockRPurchaseEntry) {
				m.EXPECT().CreatePurchaseEntry(gomock.Any(), gomock.Any(), entry).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRPurchaseEntry(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := medicine.NewPurchaseEntryService(repo)
			err := svc.CreatePurchaseEntry(servicetest.NopLogger(), nil, entry, medicine.Net30)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
