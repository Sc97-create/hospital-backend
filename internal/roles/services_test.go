package roles_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/roles"
	"hospital-backend/internal/roles/mocks"

	"go.uber.org/mock/gomock"
)

func newRoleService(t *testing.T, repo roles.RoleRepository) *roles.RoleServices {
	t.Helper()
	return roles.NewRoleServices(repo)
}

func TestServiceFindMany(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*mocks.MockRoleRepository)
		wantErr   bool
		wantTotal int64
		wantLen   int
	}{
		{
			name: "find many error",
			setup: func(m *mocks.MockRoleRepository) {
				m.EXPECT().FindMany("org-1", 10, 0).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "count error",
			setup: func(m *mocks.MockRoleRepository) {
				m.EXPECT().FindMany("org-1", 10, 0).Return([]roles.Role{{ID: "r1", Name: "Doctor"}}, nil)
				m.EXPECT().Count("org-1").Return(int64(0), errors.New("count error"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func(m *mocks.MockRoleRepository) {
				m.EXPECT().FindMany("org-1", 10, 0).Return([]roles.Role{{ID: "r1", Name: "Doctor"}}, nil)
				m.EXPECT().Count("org-1").Return(int64(1), nil)
			},
			wantTotal: 1,
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockRoleRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newRoleService(t, repo)
			list, total, err := svc.FindMany("org-1", 10, 0)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tt.wantTotal || len(list) != tt.wantLen {
				t.Fatalf("got total=%d len=%d", total, len(list))
			}
			if tt.wantLen > 0 && list[0].Name != "Doctor" {
				t.Fatalf("expected Doctor role, got %q", list[0].Name)
			}
		})
	}
}

func TestServiceInsertMany(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().InsertMany(gomock.Any(), gomock.Any()).Return(errors.New("insert failed"))

		svc := newRoleService(t, repo)
		_, err := svc.InsertMany(nil, "org-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().InsertMany(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ interface{}, roleList []roles.Role) error {
				if len(roleList) != len(roles.DefaultRoleArr) {
					t.Fatalf("expected %d roles, got %d", len(roles.DefaultRoleArr), len(roleList))
				}
				for i, name := range roles.DefaultRoleArr {
					if roleList[i].Name != name {
						t.Fatalf("expected role %q, got %q", name, roleList[i].Name)
					}
					if roleList[i].OrganisationID != "org-1" {
						t.Fatalf("expected org-1, got %q", roleList[i].OrganisationID)
					}
				}
				return nil
			},
		)

		svc := newRoleService(t, repo)
		got, err := svc.InsertMany(nil, "org-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(roles.DefaultRoleArr) {
			t.Fatalf("expected %d roles returned", len(roles.DefaultRoleArr))
		}
	})
}

func TestServiceFindRoleByOrgID(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().FindRoleByOrgID("org-1").Return(nil, errors.New("db error"))
		svc := newRoleService(t, repo)
		_, err := svc.FindRoleByOrgID("org-1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		want := []roles.Role{
			{ID: "r1", Name: "Doctor", OrganisationID: "org-1"},
			{ID: "r2", Name: "Nurse", OrganisationID: "org-1"},
		}
		repo.EXPECT().FindRoleByOrgID("org-1").Return(want, nil)
		svc := newRoleService(t, repo)
		got, err := svc.FindRoleByOrgID("org-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0].Name != "Doctor" || got[1].Name != "Nurse" {
			t.Fatalf("unexpected roles: %+v", got)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().FindRoleByOrgID("org-empty").Return([]roles.Role{}, nil)
		svc := newRoleService(t, repo)
		got, err := svc.FindRoleByOrgID("org-empty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty slice, got %+v", got)
		}
	})
}

func TestServiceFindRoleByNames(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().FindRoleByNames("org-1", "Doctor").Return(roles.Role{}, errors.New("not found"))
		svc := newRoleService(t, repo)
		_, err := svc.FindRoleByNames("org-1", "Doctor")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().FindRoleByNames("org-1", "Doctor").Return(roles.Role{ID: "r1", Name: "Doctor"}, nil)
		svc := newRoleService(t, repo)
		got, err := svc.FindRoleByNames("org-1", "Doctor")
		if err != nil || got.ID != "r1" {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestServiceFindByID(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().FindByID("r1").Return(roles.Role{}, errors.New("not found"))
		svc := newRoleService(t, repo)
		_, err := svc.FindByID("r1")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRoleRepository(ctrl)
		repo.EXPECT().FindByID("r1").Return(roles.Role{ID: "r1", Name: "Doctor"}, nil)
		svc := newRoleService(t, repo)
		got, err := svc.FindByID("r1")
		if err != nil || got.Name != "Doctor" {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}
