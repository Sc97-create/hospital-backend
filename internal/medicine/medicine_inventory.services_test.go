package medicine_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/medicine"
	"hospital-backend/internal/medicine/mocks"
	"hospital-backend/internal/testutil/servicetest"

	"go.uber.org/mock/gomock"
)

func TestCreateMedicineInventory(t *testing.T) {
	inventory := []medicine.MedicineInventory{{ID: "inv-1", MedicineID: "med-1"}}

	tests := []struct {
		name    string
		setup   func(*mocks.MockRMedicineInventory)
		wantErr bool
	}{
		{
			name: "error",
			setup: func(m *mocks.MockRMedicineInventory) {
				m.EXPECT().CreateInventoryInBatch(gomock.Any(), gomock.Any(), inventory).Return(errors.New("batch failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *mocks.MockRMedicineInventory) {
				m.EXPECT().CreateInventoryInBatch(gomock.Any(), gomock.Any(), inventory).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRMedicineInventory(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := medicine.NewSMedicineInventory(repo)
			err := svc.CreateMedicineInventory(servicetest.NopLogger(), nil, inventory)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateMedInventoryStock(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockRMedicineInventory)
		wantErr bool
	}{
		{
			name: "error",
			setup: func(m *mocks.MockRMedicineInventory) {
				m.EXPECT().UpdateMedInventoryStock(gomock.Any(), gomock.Any(), "inv-1", int64(2)).Return(errors.New("update failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *mocks.MockRMedicineInventory) {
				m.EXPECT().UpdateMedInventoryStock(gomock.Any(), gomock.Any(), "inv-1", int64(2)).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRMedicineInventory(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := medicine.NewSMedicineInventory(repo)
			err := svc.UpdateMedInventoryStock(servicetest.NopLogger(), nil, "inv-1", 2)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
