package rolepermissions_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/internal/rolepermissions/mocks"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"github.com/lib/pq"
	"go.uber.org/mock/gomock"
)

func newRolePermService(t *testing.T, repo rolepermissions.RolePermissionRepo) *rolepermissions.RolePermissionService {
	t.Helper()
	return rolepermissions.NewRolePermissionService(nil, repo)
}

func TestServiceCreate(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRolePermissionRepo(ctrl)
		rp := &rolepermissions.RolePermission{RoleID: "role-1"}
		repo.EXPECT().Create(rp).Return(errors.New("db error"))

		svc := newRolePermService(t, repo)
		if err := svc.Create(rp); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRolePermissionRepo(ctrl)
		rp := &rolepermissions.RolePermission{RoleID: "role-1"}
		repo.EXPECT().Create(rp).Return(nil)

		svc := newRolePermService(t, repo)
		if err := svc.Create(rp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceFindModulesByRoleID(t *testing.T) {
	tests := []struct {
		name       string
		roleID     string
		setup      func(*mocks.MockRolePermissionRepo)
		wantErr    error
		wantAdmin  bool
		wantPerms  int
	}{
		{
			name:      "empty role id",
			roleID:    "",
			wantAdmin: false,
			wantPerms: 0,
		},
		{
			name:   "is admin check error",
			roleID: "role-1",
			setup: func(m *mocks.MockRolePermissionRepo) {
				m.EXPECT().IsAdminRole("role-1").Return(false, errors.New("db error"))
			},
			wantErr: wrapError.ErrRolePermissionsFetchFailed,
		},
		{
			name:   "admin role",
			roleID: "role-admin",
			setup: func(m *mocks.MockRolePermissionRepo) {
				m.EXPECT().IsAdminRole("role-admin").Return(true, nil)
			},
			wantAdmin: true,
			wantPerms: 0,
		},
		{
			name:   "module permissions error",
			roleID: "role-1",
			setup: func(m *mocks.MockRolePermissionRepo) {
				m.EXPECT().IsAdminRole("role-1").Return(false, nil)
				m.EXPECT().FindModulePermissionsByRoleID("role-1").Return(nil, errors.New("query failed"))
			},
			wantErr: wrapError.ErrRolePermissionsFetchFailed,
		},
		{
			name:   "success with permissions",
			roleID: "role-1",
			setup: func(m *mocks.MockRolePermissionRepo) {
				m.EXPECT().IsAdminRole("role-1").Return(false, nil)
				m.EXPECT().FindModulePermissionsByRoleID("role-1").Return([]dto.ModulePermissionRow{
					{ModuleName: "patient", PermissionNames: pq.StringArray{"view", "create"}},
					{ModuleName: "appointment", PermissionNames: pq.StringArray{"view"}},
				}, nil)
			},
			wantAdmin: false,
			wantPerms: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRolePermissionRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newRolePermService(t, repo)
			got, err := svc.FindModulesByRoleID(tt.roleID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IsAdmin != tt.wantAdmin {
				t.Fatalf("expected is_admin=%v, got %v", tt.wantAdmin, got.IsAdmin)
			}
			if len(got.Permissions) != tt.wantPerms {
				t.Fatalf("expected %d permissions, got %d", tt.wantPerms, len(got.Permissions))
			}
			if tt.wantPerms > 0 {
				if got.Permissions[0].ModuleName != "patient" {
					t.Fatalf("expected patient module, got %q", got.Permissions[0].ModuleName)
				}
				if !got.Permissions[0].Permissions.View || !got.Permissions[0].Permissions.Create {
					t.Fatalf("expected view+create flags, got %+v", got.Permissions[0].Permissions)
				}
			}
		})
	}
}

func TestServiceInsertMany(t *testing.T) {
	roleArr := []roles.Role{
		{ID: "role-admin", Name: roles.DefaultRoleAdmin},
		{ID: "role-doctor", Name: roles.DefaultRoleDoctor},
		{ID: "role-nurse", Name: roles.DefaultRoleNurse},
		{ID: "role-receptionist", Name: roles.DefaultRoleReceptionist},
		{ID: "role-pharmacist", Name: roles.DefaultRolePharmacist},
	}
	permArr := []permissions.Permission{
		{ID: "p-create", Name: permissions.Create},
		{ID: "p-update", Name: permissions.Update},
		{ID: "p-view", Name: permissions.View},
		{ID: "p-delete", Name: permissions.Delete},
	}
	modArr := []modules.Modules{
		{ID: "m-patient", Name: constants.Patient},
		{ID: "m-appointment", Name: constants.Appointment},
		{ID: "m-prescription", Name: constants.Prescription},
		{ID: "m-employee", Name: constants.Employee},
		{ID: "m-medicine", Name: constants.Medicine},
		{ID: "m-billing", Name: constants.Billing},
	}

	t.Run("batch create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRolePermissionRepo(ctrl)
		repo.EXPECT().BatchCreate(gomock.Any(), gomock.Any()).Return(errors.New("batch failed"))

		svc := newRolePermService(t, repo)
		if err := svc.InsertMany(nil, roleArr, permArr, modArr, "org-1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRolePermissionRepo(ctrl)
		repo.EXPECT().BatchCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ interface{}, rows []rolepermissions.RolePermission) error {
				if len(rows) == 0 {
					t.Fatal("expected seeded role permissions")
				}
				var hasAdmin bool
				for _, row := range rows {
					if row.OrganisationID != "org-1" {
						t.Fatalf("expected org-1, got %q", row.OrganisationID)
					}
					if row.IsAdmin {
						hasAdmin = true
						if row.PermissionID != nil || row.ModuleID != nil {
							t.Fatal("admin row should have nil permission/module ids")
						}
					}
				}
				if !hasAdmin {
					t.Fatal("expected admin role permission row")
				}
				return nil
			},
		)

		svc := newRolePermService(t, repo)
		if err := svc.InsertMany(nil, roleArr, permArr, modArr, "org-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
