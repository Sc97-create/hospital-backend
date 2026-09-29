package organisations_test

import (
	"errors"
	"testing"
	"time"

	"hospital-backend/central/organisations"
	dto "hospital-backend/central/organisations/dto"
	"hospital-backend/central/organisations/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func validAddPayload(tenantID string) dto.OrganisationPayload {
	expiry := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	return dto.OrganisationPayload{
		TenantID:         tenantID,
		LegalEntityName:  "Acme Health Pvt Ltd",
		OrganisationType: "hospital",
		FacilityName:     "Acme City Hospital - Main Campus",
		RegistrationNo:   "REG-KA-123",
		LicenseNumber:    "CL-KA-456",
		LicenseExpiry:    &expiry,
		GSTIN:            "29AAAAA0000A1Z5",
		CountryID:        "IN",
		State:            "KA",
		City:             "BLR",
		Status:           "active",
		PatientLookup:    true,
		LabReports:       false,
	}
}

func TestServiceAddOrganisation(t *testing.T) {
	log := servicetest.NopLogger()
	payload := validAddPayload("tenant-1")

	tests := []struct {
		name    string
		payload dto.OrganisationPayload
		setup   func(*mocks.MockOrganisationRepo)
		wantErr error
		wantID  bool
	}{
		{
			name:    "missing tenant_id",
			payload: dto.OrganisationPayload{LegalEntityName: "X", OrganisationType: "hospital", FacilityName: "Y"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "missing legal_entity_name",
			payload: dto.OrganisationPayload{TenantID: "t1", OrganisationType: "hospital", FacilityName: "Y"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "whitespace only facility_name",
			payload: dto.OrganisationPayload{TenantID: "t1", LegalEntityName: "Acme", OrganisationType: "hospital", FacilityName: "   "},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "invalid organisation_type typo",
			payload: dto.OrganisationPayload{TenantID: "t1", LegalEntityName: "Acme", OrganisationType: "hospial", FacilityName: "Main"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "invalid status",
			payload: dto.OrganisationPayload{TenantID: "t1", LegalEntityName: "Acme", OrganisationType: "hospital", FacilityName: "Main", Status: "enabled"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "repo create error",
			payload: payload,
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationCreateFailed,
		},
		{
			name:    "success",
			payload: payload,
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, _ *gorm.DB, org organisations.Organisation) error {
						if org.ID == "" || org.TenantID != "tenant-1" {
							t.Fatalf("unexpected org: %+v", org)
						}
						if org.LegalEntityName != payload.LegalEntityName {
							t.Fatalf("legal_entity_name: got %q", org.LegalEntityName)
						}
						if !org.DataSharing.PatientLookup || org.DataSharing.LabReports {
							t.Fatalf("data_sharing: %+v", org.DataSharing)
						}
						return nil
					},
				)
			},
			wantID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockOrganisationRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := organisations.NewOrganisationService(nil, repo)
			id, err := svc.AddOrganisation(log, tt.payload)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				if id != "" {
					t.Fatalf("expected empty id, got %q", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tt.wantID && id == "" {
				t.Fatal("expected organisation id")
			}
		})
	}
}

func TestServiceGetOrgByID(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name    string
		orgID   string
		setup   func(*mocks.MockOrganisationRepo)
		wantErr error
		wantID  string
	}{
		{
			name:    "empty id",
			orgID:   "  ",
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:  "not found",
			orgID: "org-missing",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().GetByID(log, gomock.Any(), "org-missing").Return(organisations.Organisation{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name:  "db error",
			orgID: "org-1",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().GetByID(log, gomock.Any(), "org-1").Return(organisations.Organisation{}, errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationFetchFailed,
		},
		{
			name:  "success",
			orgID: "org-1",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().GetByID(log, gomock.Any(), "org-1").Return(organisations.Organisation{
					ID:              "org-1",
					LegalEntityName: "Acme",
				}, nil)
			},
			wantID: "org-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockOrganisationRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := organisations.NewOrganisationService(nil, repo)
			got, err := svc.GetOrgByID(log, tt.orgID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("id: got %q want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestServiceListByTenantID(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name      string
		tenantID  string
		setup     func(*mocks.MockOrganisationRepo)
		wantErr   error
		wantCount int
	}{
		{
			name:     "empty tenant",
			tenantID: "",
			wantErr:  wrapError.ErrInvalidRequest,
		},
		{
			name:     "db error",
			tenantID: "tenant-1",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().ListByTenantID(log, gomock.Any(), "tenant-1").Return(nil, errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationFetchFailed,
		},
		{
			name:     "success",
			tenantID: "tenant-1",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().ListByTenantID(log, gomock.Any(), "tenant-1").Return([]organisations.Organisation{
					{ID: "org-1"},
					{ID: "org-2"},
				}, nil)
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockOrganisationRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := organisations.NewOrganisationService(nil, repo)
			got, err := svc.ListByTenantID(log, tt.tenantID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("count: got %d want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestServiceUpdateAddress(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name    string
		payload dto.OrganisationPayload
		setup   func(*mocks.MockOrganisationRepo)
		wantErr error
	}{
		{
			name:    "missing organisation_id",
			payload: dto.OrganisationPayload{CountryID: "IN"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name: "not found",
			payload: dto.OrganisationPayload{
				OrganisationID: "org-1",
				CountryID:      "IN",
				State:          "KA",
				City:           "BLR",
				PatientLookup:  true,
			},
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().UpdateAddressByID(log, gomock.Any(), gomock.Any()).Return(gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name: "db error",
			payload: dto.OrganisationPayload{
				OrganisationID: "org-1",
				CountryID:      "IN",
				State:          "KA",
				City:           "BLR",
			},
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().UpdateAddressByID(log, gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationUpdateFailed,
		},
		{
			name: "success",
			payload: dto.OrganisationPayload{
				OrganisationID: "org-1",
				CountryID:      "IN",
				State:          "KA",
				City:           "BLR",
				PatientLookup:  true,
				LabReports:     true,
			},
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().UpdateAddressByID(log, gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, _ string, cond ...any) error {
						if len(cond) != 3 {
							t.Fatalf("expected 3 cond args, got %#v", cond)
						}
						address, ok := cond[0].(organisations.Address)
						if !ok || address.City != "BLR" {
							t.Fatalf("unexpected address: %#v", cond[0])
						}
						ds, ok := cond[1].(organisations.DataSharing)
						if !ok || !ds.PatientLookup || !ds.LabReports {
							t.Fatalf("unexpected data_sharing: %#v", cond[1])
						}
						id, ok := cond[2].(string)
						if !ok || id != "org-1" {
							t.Fatalf("unexpected id: %#v", cond[2])
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
			repo := mocks.NewMockOrganisationRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := organisations.NewOrganisationService(nil, repo)
			err := svc.UpdateAddress(log, tt.payload)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name    string
		orgID   string
		payload dto.OrganisationPayload
		setup   func(*mocks.MockOrganisationRepo)
		wantErr error
	}{
		{
			name:    "empty organisation id",
			orgID:   "",
			payload: dto.OrganisationPayload{FacilityName: "X"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "no fields to update",
			orgID:   "org-1",
			payload: dto.OrganisationPayload{},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "whitespace only fields treated as empty",
			orgID:   "org-1",
			payload: dto.OrganisationPayload{FacilityName: "   ", GSTIN: "  "},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "invalid organisation_type",
			orgID:   "org-1",
			payload: dto.OrganisationPayload{OrganisationType: "nursing_home"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "invalid status",
			orgID:   "org-1",
			payload: dto.OrganisationPayload{Status: "enabled"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:  "not found",
			orgID: "org-1",
			payload: dto.OrganisationPayload{
				FacilityName: "New Campus",
			},
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Update(log, gomock.Any(), gomock.Any()).Return(gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name:  "db error",
			orgID: "org-1",
			payload: dto.OrganisationPayload{
				FacilityName: "New Campus",
			},
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Update(log, gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationUpdateFailed,
		},
		{
			name:  "success",
			orgID: "org-1",
			payload: dto.OrganisationPayload{
				FacilityName: "New Campus",
				GSTIN:        "29BBBBB0000B1Z5",
			},
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Update(log, gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, query string, cond ...any) error {
						if len(cond) < 3 {
							t.Fatalf("expected update args, got %#v", cond)
						}
						foundFacility, foundGSTIN := false, false
						for _, c := range cond {
							if c == "New Campus" {
								foundFacility = true
							}
							if c == "29BBBBB0000B1Z5" {
								foundGSTIN = true
							}
						}
						if !foundFacility || !foundGSTIN {
							t.Fatalf("missing expected update values in %#v", cond)
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
			repo := mocks.NewMockOrganisationRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := organisations.NewOrganisationService(nil, repo)
			err := svc.Update(log, tt.orgID, tt.payload)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}
