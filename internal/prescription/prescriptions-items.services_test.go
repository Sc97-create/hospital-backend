package prescription_test

import (
	"context"
	"errors"
	"testing"

	"hospital-backend/internal/prescription"
	"hospital-backend/internal/prescription/dto"
	prescmocks "hospital-backend/internal/prescription/mocks"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newPrescriptionItemService(t *testing.T, repo prescription.PrescItemsRepo) *prescription.PrescriptionItemServ {
	t.Helper()
	return prescription.NewPrescriptionItemService(repo)
}

func TestServiceAddItems(t *testing.T) {
	log := servicetest.NopLogger()
	med := servicetest.ValidMedicineArray()

	tests := []struct {
		name    string
		med     []dto.MedicineArray
		setup   func(*prescmocks.MockPrescItemsRepo)
		wantErr error
	}{
		{
			name: "empty medicine_id",
			med: func() []dto.MedicineArray {
				m := servicetest.ValidMedicineArray()
				m.MedicineID = ""
				return []dto.MedicineArray{m}
			}(),
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetMedicineIDsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(nil, nil)
			},
			wantErr: errors.New("medicine_id is required"),
		},
		{
			name: "duplicate medicine",
			med:  []dto.MedicineArray{med},
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetMedicineIDsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return([]string{"med-1"}, nil)
			},
			wantErr: wrapError.ErrMedicineAlreadyPresent,
		},
		{
			name: "repo add error",
			med:  []dto.MedicineArray{med},
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetMedicineIDsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(nil, nil)
				m.EXPECT().AddItems(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "success",
			med:  []dto.MedicineArray{med},
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetMedicineIDsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(nil, nil)
				m.EXPECT().AddItems(gomock.Any(), gomock.Any(), gomock.Len(1)).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := prescmocks.NewMockPrescItemsRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newPrescriptionItemService(t, repo)
			err := svc.AddItems(log, nil, tt.med, "rx-1", "doc-1")
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() && !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestServiceUpdatePrescriptionItemByID(t *testing.T) {
	log := servicetest.NopLogger()
	req := servicetest.ValidPrescriptionItemUpdate()

	tests := []struct {
		name    string
		setup   func(*prescmocks.MockPrescItemsRepo)
		wantErr error
	}{
		{
			name: "not found",
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetPrescriptionItemByID(gomock.Any(), "pi-1").Return(prescription.PrescriptionItems{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrPrescriptionItemNotFound,
		},
		{
			name: "already dispensed",
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetPrescriptionItemByID(gomock.Any(), "pi-1").Return(prescription.PrescriptionItems{
					ID: "pi-1", Quantity: 10, BalanceAfterDispense: 5, Status: constants.StatusPending,
				}, nil)
			},
			wantErr: wrapError.ErrCannotEditDispensedItem,
		},
		{
			name: "success",
			setup: func(m *prescmocks.MockPrescItemsRepo) {
				m.EXPECT().GetPrescriptionItemByID(gomock.Any(), "pi-1").Return(prescription.PrescriptionItems{
					ID: "pi-1", Quantity: 7, BalanceAfterDispense: 7, Status: constants.StatusPending,
				}, nil)
				m.EXPECT().UpdatePrescriptionItem(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := prescmocks.NewMockPrescItemsRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newPrescriptionItemService(t, repo)
			err := svc.UpdatePrescriptionItemByID(log, req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestServiceGetqtyByMedicine(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().GetQtyInfoByMed(gomock.Any(), "rx-1").Return(nil, errors.New("db error"))
		svc := newPrescriptionItemService(t, repo)
		_, err := svc.GetqtyByMedicine(log, "rx-1")
		if !errors.Is(err, wrapError.ErrPrescriptionFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().GetQtyInfoByMed(gomock.Any(), "rx-1").Return([]prescription.PrescriptionItems{
			{ID: "pi-1", MedicineID: "med-1", Quantity: 7, BalanceAfterDispense: 7},
		}, nil)
		svc := newPrescriptionItemService(t, repo)
		got, err := svc.GetqtyByMedicine(log, "rx-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, ok := got["med-1"]
		if !ok || info.Quantity != 7 || info.PrescriptionID != "pi-1" {
			t.Fatalf("unexpected map: %+v", got)
		}
	})
}

func TestServiceGetPrescriptionsByPIDWithLimit(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("read error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().GetItemsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1", gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
		svc := newPrescriptionItemService(t, repo)
		_, _, err := svc.GetPrescriptionsByPIDWithLimit(log, "rx-1", 10, 1)
		if !errors.Is(err, wrapError.ErrPrescriptionFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().GetItemsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1", gomock.Any(), gomock.Any()).Return([]prescription.MixedPrescriptionItem{}, nil)
		repo.EXPECT().GetTotalCountByPrescID(gomock.Any(), "rx-1").Return(int64(0), errors.New("count error"))
		svc := newPrescriptionItemService(t, repo)
		_, _, err := svc.GetPrescriptionsByPIDWithLimit(log, "rx-1", 10, 1)
		if !errors.Is(err, wrapError.ErrPrescriptionFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		items := []prescription.MixedPrescriptionItem{{PrescriptionItemID: "pi-1", MedicineID: "med-1"}}
		repo.EXPECT().GetItemsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1", 10, 0).Return(items, nil)
		repo.EXPECT().GetTotalCountByPrescID(gomock.Any(), "rx-1").Return(int64(1), nil)
		svc := newPrescriptionItemService(t, repo)
		got, total, err := svc.GetPrescriptionsByPIDWithLimit(log, "rx-1", 10, 1)
		if err != nil || len(got) != 1 || total != 1 {
			t.Fatalf("got items=%+v total=%d err=%v", got, total, err)
		}
	})
}

func TestServiceGetMedicineInfo(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().FindMedicineInfoByPID(gomock.Any(), context.TODO(), gomock.Any(), "rx-1").Return(nil, errors.New("db error"))
		svc := newPrescriptionItemService(t, repo)
		_, _, err := svc.GetMedicineInfo(log, "rx-1")
		if !errors.Is(err, wrapError.ErrMedicineInfoFetchFailed) {
			t.Fatalf("expected medicine info fetch failed, got %v", err)
		}
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().FindMedicineInfoByPID(gomock.Any(), context.TODO(), gomock.Any(), "rx-1").Return([]prescription.MedicineDetInfo{}, nil)
		repo.EXPECT().GetTotalCountByPrescID(gomock.Any(), "rx-1").Return(int64(0), errors.New("count error"))
		svc := newPrescriptionItemService(t, repo)
		_, _, err := svc.GetMedicineInfo(log, "rx-1")
		if !errors.Is(err, wrapError.ErrMedicineInfoFetchFailed) {
			t.Fatalf("expected medicine info fetch failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		info := []prescription.MedicineDetInfo{{MedicineID: "med-1", MedicineName: "Paracetamol"}}
		repo.EXPECT().FindMedicineInfoByPID(gomock.Any(), context.TODO(), gomock.Any(), "rx-1").Return(info, nil)
		repo.EXPECT().GetTotalCountByPrescID(gomock.Any(), "rx-1").Return(int64(1), nil)
		svc := newPrescriptionItemService(t, repo)
		got, total, err := svc.GetMedicineInfo(log, "rx-1")
		if err != nil || len(got) != 1 || total != 1 {
			t.Fatalf("got info=%+v total=%d err=%v", got, total, err)
		}
	})
}

func TestServiceUpdateDispenseItemQty(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().UpdateDispenseItemQty(gomock.Any(), gomock.Any(), gomock.Any(), "pi-1", int64(2)).Return(errors.New("db error"))
		svc := newPrescriptionItemService(t, repo)
		err := svc.UpdateDispenseItemQty(log, nil, "pi-1", 2)
		if !errors.Is(err, wrapError.ErrPrescriptionItemUpdateFailed) {
			t.Fatalf("expected update failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().UpdateDispenseItemQty(gomock.Any(), gomock.Any(), gomock.Any(), "pi-1", int64(2)).Return(nil)
		svc := newPrescriptionItemService(t, repo)
		if err := svc.UpdateDispenseItemQty(log, nil, "pi-1", 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceUpdateIPrescriptionStatus(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().UpdatePrescriptionItemStatus(gomock.Any(), gomock.Any(), "pi-1", constants.StatusFullyDispensed, false).Return(errors.New("db error"))
		svc := newPrescriptionItemService(t, repo)
		err := svc.UpdateIPrescriptionStatus(log, nil, "pi-1", constants.StatusFullyDispensed, false)
		if !errors.Is(err, wrapError.ErrPrescriptionItemUpdateFailed) {
			t.Fatalf("expected update failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescItemsRepo(ctrl)
		repo.EXPECT().UpdatePrescriptionItemStatus(gomock.Any(), gomock.Any(), "pi-1", constants.StatusFullyDispensed, false).Return(nil)
		svc := newPrescriptionItemService(t, repo)
		if err := svc.UpdateIPrescriptionStatus(log, nil, "pi-1", constants.StatusFullyDispensed, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
