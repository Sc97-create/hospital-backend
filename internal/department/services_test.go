package department_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/department"
	"hospital-backend/internal/department/mocks"

	"go.uber.org/mock/gomock"
)

func newDeptService(t *testing.T, repo department.DepartmentRepository) *department.DepartmentService {
	t.Helper()
	return department.NewDepartmentService(repo)
}

func TestServiceFindMany(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDepartmentRepository(ctrl)

	tests := []struct {
		name      string
		setup     func()
		wantErr   bool
		wantTotal int64
	}{
		{
			name: "find many error",
			setup: func() {
				repo.EXPECT().FindMany("org-1", 10, 0).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "count error",
			setup: func() {
				repo.EXPECT().FindMany("org-1", 10, 0).Return([]department.Department{{ID: "d1"}}, nil)
				repo.EXPECT().Count("org-1").Return(int64(0), errors.New("count error"))
			},
			wantErr: true,
		},
		{
			name: "success",
			setup: func() {
				repo.EXPECT().FindMany("org-1", 10, 0).Return([]department.Department{{ID: "d1"}}, nil)
				repo.EXPECT().Count("org-1").Return(int64(1), nil)
			},
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			svc := newDeptService(t, repo)
			depts, total, err := svc.FindMany("org-1", 10, 0)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(depts) != 1 || total != tt.wantTotal {
					t.Fatalf("got %d depts, total %d", len(depts), total)
				}
			}
		})
	}
}

func TestServiceInsertMany(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("batch insert error", func(t *testing.T) {
		repo := mocks.NewMockDepartmentRepository(ctrl)
		repo.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).Return(errors.New("insert error"))
		svc := newDeptService(t, repo)
		if err := svc.InsertMany(nil, "org-1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockDepartmentRepository(ctrl)
		repo.EXPECT().BatchInsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ interface{}, depts []department.Department) error {
			if len(depts) != 8 {
				t.Fatalf("expected 8 departments, got %d", len(depts))
			}
			return nil
		})
		svc := newDeptService(t, repo)
		if err := svc.InsertMany(nil, "org-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceFindDeptByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDepartmentRepository(ctrl)

	t.Run("repo error", func(t *testing.T) {
		repo.EXPECT().FindDeptByName("org-1", "Administration").Return(department.Department{}, errors.New("not found"))
		svc := newDeptService(t, repo)
		if _, err := svc.FindDeptByName("org-1", "Administration"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		want := department.Department{ID: "d1", Name: "Administration"}
		repo.EXPECT().FindDeptByName("org-1", "Administration").Return(want, nil)
		svc := newDeptService(t, repo)
		got, err := svc.FindDeptByName("org-1", "Administration")
		if err != nil || got.ID != want.ID {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestServiceFindByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDepartmentRepository(ctrl)

	t.Run("repo error", func(t *testing.T) {
		repo.EXPECT().FindByID("d1").Return(department.Department{}, errors.New("not found"))
		svc := newDeptService(t, repo)
		if _, err := svc.FindByID("d1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		want := department.Department{ID: "d1", Name: "Pharmacy"}
		repo.EXPECT().FindByID("d1").Return(want, nil)
		svc := newDeptService(t, repo)
		got, err := svc.FindByID("d1")
		if err != nil || got.Name != want.Name {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}
