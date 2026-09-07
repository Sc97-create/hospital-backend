package license_test

import (
	"errors"
	"testing"
	"time"

	"hospital-backend/internal/license"
	"hospital-backend/internal/license/mocks"
	"hospital-backend/internal/license/utils"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newLicenseService(t *testing.T, repo license.LicenseRepository) *license.LicenseService {
	t.Helper()
	return license.NewLicenseService(repo)
}

func TestServiceVerifyLicense(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	issuedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	key, _ := utils.GenerateLicenseKey("Acme Hospital", 1, "org-1", "month", issuedAt)

	tests := []struct {
		name    string
		orgID   string
		key     string
		setup   func(*mocks.MockLicenseRepository)
		wantErr error
	}{
		{
			name:    "empty org id",
			orgID:   "",
			key:     key,
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "empty key",
			orgID:   "org-1",
			key:     "",
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:  "not found",
			orgID: "org-1",
			key:   key,
			setup: func(m *mocks.MockLicenseRepository) {
				m.EXPECT().GetLicense(log, "org-1", key).Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrLicenseNotFound,
		},
		{
			name:  "db error",
			orgID: "org-1",
			key:   key,
			setup: func(m *mocks.MockLicenseRepository) {
				m.EXPECT().GetLicense(log, "org-1", key).Return(nil, errors.New("db error"))
			},
			wantErr: wrapError.ErrLicenseVerifyFailed,
		},
		{
			name:  "key mismatch",
			orgID: "org-1",
			key:   key,
			setup: func(m *mocks.MockLicenseRepository) {
				otherKey, _ := utils.GenerateLicenseKey("Acme Hospital", 1, "org-1", "year", issuedAt)
				m.EXPECT().GetLicense(log, "org-1", key).Return(&license.License{ID: "lic-1", LicenseKey: otherKey}, nil)
			},
			wantErr: wrapError.ErrLicenseInvalid,
		},
		{
			name:  "success",
			orgID: "org-1",
			key:   key,
			setup: func(m *mocks.MockLicenseRepository) {
				m.EXPECT().GetLicense(log, "org-1", key).Return(&license.License{ID: "lic-1", LicenseKey: key}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockLicenseRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newLicenseService(t, repo)
			err := svc.VerifyLicense(log, tt.orgID, tt.key)
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

func TestServiceCreateLicenseSrv(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	issuedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("repo create error", func(t *testing.T) {
		repo := mocks.NewMockLicenseRepository(ctrl)
		repo.EXPECT().CreateLicense(log, gomock.Any(), gomock.Any()).Return(errors.New("create error"))
		svc := newLicenseService(t, repo)
		err := svc.CreateLicenseSrv(log, nil, "Acme", 1, "org-1", "month", issuedAt)
		if !errors.Is(err, wrapError.ErrLicenseCreateFailed) {
			t.Fatalf("expected create failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockLicenseRepository(ctrl)
		repo.EXPECT().CreateLicense(log, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ interface{}, _ interface{}, lic *license.License) error {
				if lic.OrganisationID != "org-1" {
					t.Fatalf("expected org-1, got %q", lic.OrganisationID)
				}
				if lic.LicenseKey == "" || lic.ID == "" {
					t.Fatal("expected generated license key and id")
				}
				return nil
			},
		)
		svc := newLicenseService(t, repo)
		if err := svc.CreateLicenseSrv(log, nil, "Acme", 1, "org-1", "month", issuedAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
