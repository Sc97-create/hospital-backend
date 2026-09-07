package permissions_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/modules"
	modmocks "hospital-backend/internal/modules/mocks"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/permissions/mocks"

	"go.uber.org/mock/gomock"
)

func newPermService(t *testing.T, permRepo permissions.PermissionRepo, moduleRepo modules.ModuleRepo) *permissions.PermService {
	t.Helper()
	return permissions.NewService(permRepo, moduleRepo)
}

func TestServiceFindMany(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockPermissionRepo, *modmocks.MockModuleRepo)
		wantErr bool
		wantMod int
		wantPerm int
	}{
		{
			name: "permission repo error",
			setup: func(perm *mocks.MockPermissionRepo, _ *modmocks.MockModuleRepo) {
				perm.EXPECT().FindMany().Return(nil, errors.New("perm error"))
			},
			wantErr: true,
		},
		{
			name: "module repo error",
			setup: func(perm *mocks.MockPermissionRepo, mod *modmocks.MockModuleRepo) {
				perm.EXPECT().FindMany().Return([]permissions.Permission{{ID: "p1"}}, nil)
				mod.EXPECT().FindMany().Return(nil, errors.New("module error"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(perm *mocks.MockPermissionRepo, mod *modmocks.MockModuleRepo) {
				perm.EXPECT().FindMany().Return([]permissions.Permission{{ID: "p1", Name: "view"}}, nil)
				mod.EXPECT().FindMany().Return([]modules.Modules{{ID: "m1", Name: "patient"}}, nil)
			},
			wantMod:  1,
			wantPerm: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			permRepo := mocks.NewMockPermissionRepo(ctrl)
			modRepo := modmocks.NewMockModuleRepo(ctrl)
			if tt.setup != nil {
				tt.setup(permRepo, modRepo)
			}

			svc := newPermService(t, permRepo, modRepo)
			mods, perms, err := svc.FindMany()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mods) != tt.wantMod || len(perms) != tt.wantPerm {
				t.Fatalf("got %d modules, %d permissions", len(mods), len(perms))
			}
		})
	}
}

func TestServiceDefaultPerm(t *testing.T) {
	t.Run("batch insert error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		permRepo := mocks.NewMockPermissionRepo(ctrl)
		permRepo.EXPECT().BatchInsert(gomock.Any(), 2).Return(errors.New("insert failed"))

		svc := newPermService(t, permRepo, nil)
		if err := svc.DefaultPerm(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		permRepo := mocks.NewMockPermissionRepo(ctrl)
		permRepo.EXPECT().BatchInsert(gomock.Any(), 2).DoAndReturn(
			func(perms []permissions.Permission, batchSize int) error {
				if batchSize != 2 {
					t.Fatalf("expected batch size 2, got %d", batchSize)
				}
				if len(perms) != len(permissions.AdminPermArr) {
					t.Fatalf("expected %d permissions, got %d", len(permissions.AdminPermArr), len(perms))
				}
				for i, name := range permissions.AdminPermArr {
					if perms[i].Name != name {
						t.Fatalf("expected permission %q at index %d, got %q", name, i, perms[i].Name)
					}
					if perms[i].ID == "" {
						t.Fatalf("expected generated id at index %d", i)
					}
				}
				return nil
			},
		)

		svc := newPermService(t, permRepo, nil)
		if err := svc.DefaultPerm(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
