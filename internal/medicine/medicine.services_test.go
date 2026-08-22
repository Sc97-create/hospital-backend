package medicine_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/medicine"
	meddto "hospital-backend/internal/medicine/dto"
	"hospital-backend/internal/medicine/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newMedicineService(t *testing.T, repo medicine.MedicineRepository) *medicine.MedicineService {
	t.Helper()
	return medicine.NewMedicineService(nil, repo, nil, nil, nil, nil)
}

func TestGetOne(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockMedicineRepository)
		wantErr error
		wantID  string
	}{
		{
			name: "not found",
			setup: func(m *mocks.MockMedicineRepository) {
				m.EXPECT().FindOne(gomock.Any(), "med-missing").Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrMedicineNotFound,
		},
		{
			name: "success",
			setup: func(m *mocks.MockMedicineRepository) {
				m.EXPECT().FindOne(gomock.Any(), "med-1").Return(&medicine.Medicine{
					ID:   "med-1",
					Name: "Paracetamol",
				}, nil)
			},
			wantID: "med-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMedicineRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newMedicineService(t, repo)
			id := "med-1"
			if tt.wantErr == wrapError.ErrMedicineNotFound {
				id = "med-missing"
			}
			got, err := svc.GetOne(servicetest.NopLogger(), id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("expected id %q, got %q", tt.wantID, got.ID)
			}
		})
	}
}

func TestGetMany(t *testing.T) {
	meds := []medicine.Medicine{{ID: "med-1", Name: "Paracetamol"}}

	tests := []struct {
		name    string
		setup   func(*mocks.MockMedicineRepository)
		wantErr bool
		wantLen int
	}{
		{
			name: "repo error",
			setup: func(m *mocks.MockMedicineRepository) {
				m.EXPECT().FindMany(gomock.Any(), "1=1 LIMIT ? OFFSET ?", 10, 0).Return(nil, errors.New("read failed"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *mocks.MockMedicineRepository) {
				m.EXPECT().FindMany(gomock.Any(), "1=1 LIMIT ? OFFSET ?", 10, 0).Return(meds, nil)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMedicineRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newMedicineService(t, repo)
			got, err := svc.GetMany(servicetest.NopLogger(), 10, 1)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Fatalf("expected %d medicines, got %d", tt.wantLen, len(got))
			}
		})
	}
}

func TestMedicineServiceSearchMedicine(t *testing.T) {
	results := []meddto.SearchMedicineItem{{ID: "med-1", Name: "Paracetamol"}}

	tests := []struct {
		name    string
		setup   func(*mocks.MockMedicineRepository)
		wantErr error
		wantLen int
	}{
		{
			name: "repo error",
			setup: func(m *mocks.MockMedicineRepository) {
				m.EXPECT().SearchMedicine(gomock.Any(), "para", "para%", "org-1").Return(nil, errors.New("search failed"))
			},
			wantErr: wrapError.ErrMedicineSearchFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockMedicineRepository) {
				m.EXPECT().SearchMedicine(gomock.Any(), "para", "para%", "org-1").Return(results, nil)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMedicineRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newMedicineService(t, repo)
			got, err := svc.SearchMedicine(servicetest.NopLogger(), "para", "org-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(got))
			}
		})
	}
}

func TestFindNamesByIds(t *testing.T) {
	ids := []string{"med-1", "med-2"}
	expected := []medicine.Medicine{
		{ID: "med-1", Name: "Paracetamol"},
		{ID: "med-2", Name: "Ibuprofen"},
	}

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMedicineRepository(ctrl)
	repo.EXPECT().FindNamesByIds(gomock.Any(), ids).Return(expected, nil)

	svc := newMedicineService(t, repo)
	got, err := svc.FindNamesByIds(servicetest.NopLogger(), ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d medicines, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i].ID != expected[i].ID || got[i].Name != expected[i].Name {
			t.Fatalf("unexpected passthrough at index %d: %+v", i, got[i])
		}
	}
}
