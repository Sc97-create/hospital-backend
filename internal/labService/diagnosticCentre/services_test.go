package diagnosticcentre_test

import (
	"errors"
	"testing"

	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	"hospital-backend/internal/labService/diagnosticCentre/dto"
	"hospital-backend/internal/labService/diagnosticCentre/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newService(t *testing.T, repo diagnosticcentre.DiagnosticCentreRepository) *diagnosticcentre.DiagnosticCentreService {
	t.Helper()
	return diagnosticcentre.NewDiagnosticCentreService(repo)
}

func TestDiagnosticCentreServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.CreateDiagnosticCentreRequest
		setup   func(*mocks.MockDiagnosticCentreRepository)
		wantErr error
	}{
		{
			name:    "missing org",
			req:     dto.CreateDiagnosticCentreRequest{Name: "City Lab"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "missing name",
			req:     dto.CreateDiagnosticCentreRequest{OrganisationID: "org-1"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name: "repo error",
			req:  dto.CreateDiagnosticCentreRequest{OrganisationID: "org-1", Name: "City Lab"},
			setup: func(m *mocks.MockDiagnosticCentreRepository) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("insert failed"))
			},
			wantErr: wrapError.ErrDiagnosticCentreCreateFailed,
		},
		{
			name: "success",
			req: dto.CreateDiagnosticCentreRequest{
				OrganisationID: "org-1",
				Name:           "  City Lab  ",
				Address:        " Main St ",
				ContactNumber:  " 999 ",
				Email:          " a@b.com ",
			},
			setup: func(m *mocks.MockDiagnosticCentreRepository) {
				m.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&diagnosticcentre.DiagnosticCentre{})).DoAndReturn(
					func(_ interface{}, row *diagnosticcentre.DiagnosticCentre) error {
						if row.Name != "City Lab" || row.Address != "Main St" || row.ContactNumber != "999" || row.Email != "a@b.com" {
							t.Fatalf("unexpected trimmed fields: %+v", row)
						}
						if row.Status != diagnosticcentre.StatusActive {
							t.Fatalf("expected active, got %q", row.Status)
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
			repo := mocks.NewMockDiagnosticCentreRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newService(t, repo)
			id, err := svc.Create(servicetest.NopLogger(), tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil || id == "" {
				t.Fatalf("expected id, got %q err=%v", id, err)
			}
		})
	}
}

func TestDiagnosticCentreServiceList(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDiagnosticCentreRepository(ctrl)
	svc := newService(t, repo)

	t.Run("missing org", func(t *testing.T) {
		_, _, err := svc.List(servicetest.NopLogger(), dto.ListDiagnosticCentresRequest{})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		_, _, err := svc.List(servicetest.NopLogger(), dto.ListDiagnosticCentresRequest{
			OrganisationID: "org-1",
			Status:         "bogus",
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]diagnosticcentre.DiagnosticCentre{
			{ID: "dc-1", OrganisationID: "org-1", Name: "City Lab", Status: diagnosticcentre.StatusActive},
		}, nil)
		repo.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)

		rows, total, err := svc.List(servicetest.NopLogger(), dto.ListDiagnosticCentresRequest{
			OrganisationID: "org-1",
			Limit:          10,
			PageNo:         1,
		})
		if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != "dc-1" {
			t.Fatalf("got rows=%+v total=%d err=%v", rows, total, err)
		}
	})
}

func TestDiagnosticCentreServiceUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDiagnosticCentreRepository(ctrl)
	svc := newService(t, repo)

	t.Run("invalid", func(t *testing.T) {
		err := svc.Update(servicetest.NopLogger(), "dc-1", dto.UpdateDiagnosticCentreRequest{})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), "City", "Addr", "1", "a@b.com", gomock.Any(), "missing").Return(gorm.ErrRecordNotFound)
		err := svc.Update(servicetest.NopLogger(), "missing", dto.UpdateDiagnosticCentreRequest{
			Name: "City", Address: "Addr", ContactNumber: "1", Email: "a@b.com",
		})
		if !errors.Is(err, wrapError.ErrDiagnosticCentreNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), "City Lab", "", "", "", gomock.Any(), "dc-1").Return(nil)
		if err := svc.Update(servicetest.NopLogger(), "dc-1", dto.UpdateDiagnosticCentreRequest{Name: "City Lab"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDiagnosticCentreServiceUpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDiagnosticCentreRepository(ctrl)
	svc := newService(t, repo)

	t.Run("invalid status", func(t *testing.T) {
		err := svc.UpdateStatus(servicetest.NopLogger(), "dc-1", "paused")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success deactivate", func(t *testing.T) {
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), diagnosticcentre.StatusInactive, gomock.Any(), "dc-1").Return(nil)
		if err := svc.UpdateStatus(servicetest.NopLogger(), "dc-1", "INACTIVE"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
