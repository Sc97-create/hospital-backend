package organisation_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/modules"
	"hospital-backend/internal/organisation"
	orgdto "hospital-backend/internal/organisation/DTO"
	"hospital-backend/internal/organisation/mocks"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/roles"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testOrganisationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func newOrganisationService(
	t *testing.T,
	db *gorm.DB,
	repo organisation.OrganisationRepo,
	perm organisation.PermissionCatalogLookup,
	license organisation.LicenseCreator,
	roleSeeder organisation.RoleSeeder,
	deptSeeder organisation.DepartmentSeeder,
	rolePermSeeder organisation.RolePermissionSeeder,
) *organisation.OrganisationService {
	t.Helper()
	return organisation.NewOrganisationService(db, repo, license, roleSeeder, deptSeeder, perm, rolePermSeeder)
}

var (
	testModules     = []modules.Modules{{ID: "mod-1", Name: "patients"}}
	testPermissions = []permissions.Permission{{ID: "perm-1", Name: "view"}}
	testRoles       = []roles.Role{{ID: "role-1", Name: "admin"}}
)

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
			name:  "not found",
			orgID: "org-missing",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().GetOrganisationByID(log, "org-missing").Return(organisation.Organisation{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name:  "db error",
			orgID: "org-1",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().GetOrganisationByID(log, "org-1").Return(organisation.Organisation{}, errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationFetchFailed,
		},
		{
			name:  "success",
			orgID: "org-1",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().GetOrganisationByID(log, "org-1").Return(organisation.Organisation{ID: "org-1", OrganisationName: "City Hospital"}, nil)
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
			svc := newOrganisationService(t, nil, repo, nil, nil, nil, nil, nil)
			got, err := svc.GetOrgByID(log, tt.orgID)
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

func TestServiceUpdate(t *testing.T) {
	log := servicetest.NopLogger()
	payload := servicetest.ValidOrganisationPayload()

	tests := []struct {
		name    string
		setup   func(*mocks.MockOrganisationRepo)
		wantErr error
	}{
		{
			name: "not found",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Update(log, "org-1", gomock.Any()).Return(gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name: "db error",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Update(log, "org-1", gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationUpdateFailed,
		},
		{
			name: "success partial fields",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().Update(log, "org-1", gomock.Any()).Return(nil)
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
			svc := newOrganisationService(t, nil, repo, nil, nil, nil, nil, nil)
			err := svc.Update(log, "org-1", orgdto.OrganisationPayload{
				OrganisationName: payload.OrganisationName,
				HospitalType:     payload.HospitalType,
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestServiceUpdateOrganisationLoc(t *testing.T) {
	log := servicetest.NopLogger()
	payload := servicetest.ValidOrganisationPayload()
	payload.OrganisationID = "org-1"
	payload.Country = "co-1"
	payload.State = "st-1"
	payload.City = "ci-1"
	payload.AuditLogs = true
	payload.EmergencyAcess = false

	tests := []struct {
		name    string
		setup   func(*mocks.MockOrganisationRepo)
		wantErr error
	}{
		{
			name: "not found",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().UpdateLocationByID(log, gomock.Any()).Return(gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name: "db error",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().UpdateLocationByID(log, gomock.Any()).Return(errors.New("db error"))
			},
			wantErr: wrapError.ErrOrganisationUpdateFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockOrganisationRepo) {
				m.EXPECT().UpdateLocationByID(log, gomock.Any()).DoAndReturn(
					func(_ interface{}, org *organisation.Organisation) error {
						if org.ID != "org-1" {
							t.Errorf("expected org id org-1, got %q", org.ID)
						}
						if org.Address.CityID != "ci-1" || org.Security.EnableAuditLog != true {
							t.Errorf("unexpected location/security mapping: %+v", org)
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
			svc := newOrganisationService(t, nil, repo, nil, nil, nil, nil, nil)
			err := svc.UpdateOrganisationLoc(log, payload)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestServiceCreateOrganisation(t *testing.T) {
	log := servicetest.NopLogger()
	payload := servicetest.ValidOrganisationPayload()

	t.Run("permissions lookup error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		perm.EXPECT().FindMany().Return(nil, nil, errors.New("perm error"))

		svc := newOrganisationService(t, testOrganisationDB(t), nil, perm, nil, nil, nil, nil)
		id, err := svc.CreateOrganisation(log, payload)
		if !errors.Is(err, wrapError.ErrOrganisationCreateFailed) || id != "" {
			t.Fatalf("got id=%q err=%v", id, err)
		}
	})

	t.Run("repo create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		repo := mocks.NewMockOrganisationRepo(ctrl)
		perm.EXPECT().FindMany().Return(testModules, testPermissions, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("create failed"))

		svc := newOrganisationService(t, testOrganisationDB(t), repo, perm, nil, nil, nil, nil)
		id, err := svc.CreateOrganisation(log, payload)
		if !errors.Is(err, wrapError.ErrOrganisationCreateFailed) || id != "" {
			t.Fatalf("got id=%q err=%v", id, err)
		}
	})

	t.Run("license create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		repo := mocks.NewMockOrganisationRepo(ctrl)
		license := mocks.NewMockLicenseCreator(ctrl)
		perm.EXPECT().FindMany().Return(testModules, testPermissions, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		license.EXPECT().CreateLicenseSrv(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("license failed"))

		svc := newOrganisationService(t, testOrganisationDB(t), repo, perm, license, nil, nil, nil)
		id, err := svc.CreateOrganisation(log, payload)
		if !errors.Is(err, wrapError.ErrOrganisationCreateFailed) || id != "" {
			t.Fatalf("got id=%q err=%v", id, err)
		}
	})

	t.Run("roles seed error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		repo := mocks.NewMockOrganisationRepo(ctrl)
		license := mocks.NewMockLicenseCreator(ctrl)
		roleSeeder := mocks.NewMockRoleSeeder(ctrl)
		perm.EXPECT().FindMany().Return(testModules, testPermissions, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		license.EXPECT().CreateLicenseSrv(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		roleSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(nil, errors.New("roles failed"))

		svc := newOrganisationService(t, testOrganisationDB(t), repo, perm, license, roleSeeder, nil, nil)
		id, err := svc.CreateOrganisation(log, payload)
		if !errors.Is(err, wrapError.ErrOrganisationCreateFailed) || id != "" {
			t.Fatalf("got id=%q err=%v", id, err)
		}
	})

	t.Run("departments seed error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		repo := mocks.NewMockOrganisationRepo(ctrl)
		license := mocks.NewMockLicenseCreator(ctrl)
		roleSeeder := mocks.NewMockRoleSeeder(ctrl)
		deptSeeder := mocks.NewMockDepartmentSeeder(ctrl)
		perm.EXPECT().FindMany().Return(testModules, testPermissions, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		license.EXPECT().CreateLicenseSrv(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		roleSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(testRoles, nil)
		deptSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(errors.New("depts failed"))

		svc := newOrganisationService(t, testOrganisationDB(t), repo, perm, license, roleSeeder, deptSeeder, nil)
		id, err := svc.CreateOrganisation(log, payload)
		if !errors.Is(err, wrapError.ErrOrganisationCreateFailed) || id != "" {
			t.Fatalf("got id=%q err=%v", id, err)
		}
	})

	t.Run("role permissions seed error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		repo := mocks.NewMockOrganisationRepo(ctrl)
		license := mocks.NewMockLicenseCreator(ctrl)
		roleSeeder := mocks.NewMockRoleSeeder(ctrl)
		deptSeeder := mocks.NewMockDepartmentSeeder(ctrl)
		rolePermSeeder := mocks.NewMockRolePermissionSeeder(ctrl)
		perm.EXPECT().FindMany().Return(testModules, testPermissions, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		license.EXPECT().CreateLicenseSrv(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		roleSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(testRoles, nil)
		deptSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(nil)
		rolePermSeeder.EXPECT().InsertMany(gomock.Any(), testRoles, testPermissions, testModules, gomock.Any()).Return(errors.New("rp failed"))

		svc := newOrganisationService(t, testOrganisationDB(t), repo, perm, license, roleSeeder, deptSeeder, rolePermSeeder)
		id, err := svc.CreateOrganisation(log, payload)
		if !errors.Is(err, wrapError.ErrOrganisationCreateFailed) || id != "" {
			t.Fatalf("got id=%q err=%v", id, err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		perm := mocks.NewMockPermissionCatalogLookup(ctrl)
		repo := mocks.NewMockOrganisationRepo(ctrl)
		license := mocks.NewMockLicenseCreator(ctrl)
		roleSeeder := mocks.NewMockRoleSeeder(ctrl)
		deptSeeder := mocks.NewMockDepartmentSeeder(ctrl)
		rolePermSeeder := mocks.NewMockRolePermissionSeeder(ctrl)

		var createdOrg organisation.Organisation
		perm.EXPECT().FindMany().Return(testModules, testPermissions, nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ interface{}, _ *gorm.DB, org organisation.Organisation) error {
				createdOrg = org
				if org.OrganisationName != payload.OrganisationName {
					t.Errorf("expected org name %q, got %q", payload.OrganisationName, org.OrganisationName)
				}
				return nil
			},
		)
		license.EXPECT().CreateLicenseSrv(gomock.Any(), gomock.Any(), payload.OrganisationName, 6, gomock.Any(), "month", gomock.Any()).Return(nil)
		roleSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(testRoles, nil)
		deptSeeder.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(nil)
		rolePermSeeder.EXPECT().InsertMany(gomock.Any(), testRoles, testPermissions, testModules, gomock.Any()).Return(nil)

		svc := newOrganisationService(t, testOrganisationDB(t), repo, perm, license, roleSeeder, deptSeeder, rolePermSeeder)
		id, err := svc.CreateOrganisation(log, payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" || id != createdOrg.ID {
			t.Fatalf("expected returned id %q to match created org %q", id, createdOrg.ID)
		}
	})
}
