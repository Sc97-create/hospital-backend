package medicine_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/medicine"
	"hospital-backend/internal/medicine/mocks"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/types"

	"go.uber.org/mock/gomock"
)

func TestCreateMedicineMvmt(t *testing.T) {
	movements := []types.MedicineStockMovements{{
		ID:         "mvmt-1",
		MedicineID: "med-1",
	}}

	tests := []struct {
		name    string
		setup   func(*mocks.MockRMedicineMvmt)
		wantErr bool
	}{
		{
			name: "error",
			setup: func(m *mocks.MockRMedicineMvmt) {
				m.EXPECT().CreateMedicineMvmtInBatch(gomock.Any(), gomock.Any(), movements).Return(errors.New("batch failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *mocks.MockRMedicineMvmt) {
				m.EXPECT().CreateMedicineMvmtInBatch(gomock.Any(), gomock.Any(), movements).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRMedicineMvmt(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := medicine.NewMedicineMvmt(repo)
			err := svc.CreateMedicineMvmt(servicetest.NopLogger(), nil, movements)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
