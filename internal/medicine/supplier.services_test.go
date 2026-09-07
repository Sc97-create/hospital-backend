package medicine_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"hospital-backend/internal/medicine"
	meddto "hospital-backend/internal/medicine/dto"
	"hospital-backend/internal/medicine/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newSupplierService(t *testing.T, repo medicine.ISupplier) *medicine.SupplierService {
	t.Helper()
	return medicine.NewSupplierService(repo)
}

func TestSupplierServiceGetSupplierByID(t *testing.T) {
	tests := []struct {
		name       string
		supplierID string
		setup      func(*mocks.MockISupplier)
		wantErr    error
		wantID     string
	}{
		{
			name:       "not found",
			supplierID: "sup-missing",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByID(gomock.Any(), "sup-missing").Return(medicine.Supplier{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrSupplierNotFound,
		},
		{
			name:       "db error",
			supplierID: "sup-1",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByID(gomock.Any(), "sup-1").Return(medicine.Supplier{}, errors.New("db down"))
			},
			wantErr: wrapError.ErrSupplierFetchFailed,
		},
		{
			name:       "success",
			supplierID: "sup-1",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByID(gomock.Any(), "sup-1").Return(medicine.Supplier{
					ID:   "sup-1",
					Name: "Acme Pharma",
				}, nil)
			},
			wantID: "sup-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockISupplier(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newSupplierService(t, repo)
			got, err := svc.GetSupplierByID(servicetest.NopLogger(), tt.supplierID)
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

func TestCretateSupplier(t *testing.T) {
	req := servicetest.ValidSupplierCreateRequest()

	tests := []struct {
		name    string
		setup   func(*mocks.MockISupplier)
		wantErr error
	}{
		{
			name: "repo error",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().CretateSupplier(gomock.Any(), gomock.Any()).Return(errors.New("insert failed"))
			},
			wantErr: wrapError.ErrSupplierCreateFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().CretateSupplier(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, s *medicine.Supplier) error {
						if s.Name != req.Name {
							t.Errorf("expected name %q, got %q", req.Name, s.Name)
						}
						if s.OrganisationID != req.OrganisationID {
							t.Errorf("expected org %q, got %q", req.OrganisationID, s.OrganisationID)
						}
						if s.PaymentTerms != medicine.Net30 {
							t.Errorf("expected Net 30 payment terms, got %q", s.PaymentTerms)
						}
						if s.SupplierStatus != medicine.Active {
							t.Errorf("expected Active status, got %q", s.SupplierStatus)
						}
						if !strings.HasPrefix(s.SupplierCode, "SUPP-") {
							t.Errorf("expected SUPP- prefix, got %q", s.SupplierCode)
						}
						if s.CreatedBy != req.UserID {
							t.Errorf("expected created_by %q, got %q", req.UserID, s.CreatedBy)
						}
						return nil
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockISupplier(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newSupplierService(t, repo)
			id, err := svc.CretateSupplier(servicetest.NopLogger(), req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if id != "" {
					t.Fatalf("expected empty id on error, got %q", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id == "" {
				t.Fatal("expected generated supplier id")
			}
		})
	}
}

func TestSupplierServiceGetSupplierByOrgID(t *testing.T) {
	req := meddto.SupplierListReq{
		OrganisationID: "org-1",
		Limit:          10,
		PageNo:         1,
	}
	createdAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	suppliers := []medicine.Supplier{{
		ID:           "sup-1",
		SupplierCode: "SUPP-1001",
		Name:         "Acme Pharma",
		CreatedAt:    createdAt,
	}}

	tests := []struct {
		name      string
		setup     func(*mocks.MockISupplier)
		search    string
		wantErr   error
		wantTotal int64
		wantLen   int
		wantDate  string
	}{
		{
			name: "list error",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("list failed"))
			},
			wantErr: wrapError.ErrSupplierFetchFailed,
		},
		{
			name: "count error",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(suppliers, nil)
				m.EXPECT().CountSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errors.New("count failed"))
			},
			wantErr: wrapError.ErrSupplierFetchFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(suppliers, nil)
				m.EXPECT().CountSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)
			},
			wantTotal: 1,
			wantLen:   1,
		},
		{
			name: "success with search trim",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().GetSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(suppliers, nil)
				m.EXPECT().CountSupplierByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)
			},
			search:    "  acme  ",
			wantTotal: 1,
			wantLen:   1,
			wantDate:  "15 Jan 2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockISupplier(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newSupplierService(t, repo)
			listReq := req
			if tt.search != "" {
				listReq.Search = tt.search
			}
			list, total, err := svc.GetSupplierByOrgID(servicetest.NopLogger(), listReq)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tt.wantTotal {
				t.Fatalf("expected total %d, got %d", tt.wantTotal, total)
			}
			if len(list) != tt.wantLen {
				t.Fatalf("expected list length %d, got %d", tt.wantLen, len(list))
			}
			if tt.wantLen > 0 && list[0].ID != "sup-1" {
				t.Fatalf("expected supplier id sup-1, got %q", list[0].ID)
			}
			if tt.wantDate != "" && list[0].CreatedAt != tt.wantDate {
				t.Fatalf("expected created_at %q, got %q", tt.wantDate, list[0].CreatedAt)
			}
		})
	}
}

func TestGetTotalCount(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*mocks.MockISupplier)
		wantErr   error
		wantTotal int64
	}{
		{
			name: "error",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().CountSupplierByOrgID(gomock.Any(), gomock.Any(), "org-1").Return(int64(0), errors.New("count failed"))
			},
			wantErr: wrapError.ErrSupplierFetchFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockISupplier) {
				m.EXPECT().CountSupplierByOrgID(gomock.Any(), gomock.Any(), "org-1").Return(int64(5), nil)
			},
			wantTotal: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockISupplier(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newSupplierService(t, repo)
			total, err := svc.GetTotalCount(servicetest.NopLogger(), "org-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tt.wantTotal {
				t.Fatalf("expected total %d, got %d", tt.wantTotal, total)
			}
		})
	}
}
