package billing_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/billing"
	"hospital-backend/internal/billing/dto"
	billingmocks "hospital-backend/internal/billing/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
)

func TestGetMedicineInventoryDetByInvoiceID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*billingmocks.MockInvoiceItemRepo)
		wantErr error
		wantLen int
	}{
		{
			name: "repo error",
			setup: func(m *billingmocks.MockInvoiceItemRepo) {
				m.EXPECT().GetInvoiceItemsByInvoiceID(gomock.Any(), gomock.Any(), "inv-1").Return(nil, errors.New("db"))
			},
			wantErr: wrapError.ErrInvoiceFetchFailed,
		},
		{
			name: "success",
			setup: func(m *billingmocks.MockInvoiceItemRepo) {
				m.EXPECT().GetInvoiceItemsByInvoiceID(gomock.Any(), gomock.Any(), "inv-1").Return([]dto.MedInvoiceItemResponse{
					{MedicineID: "med-1"},
				}, nil)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := billingmocks.NewMockInvoiceItemRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := billing.NewInvoiceItemServ(repo, nil)
			got, err := svc.GetMedicineInventoryDetByInvoiceID(servicetest.NopLogger(), "inv-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d items, got %d", tt.wantLen, len(got))
			}
		})
	}
}
