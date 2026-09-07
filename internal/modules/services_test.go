package modules_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/modules"
	"hospital-backend/internal/modules/mocks"

	"go.uber.org/mock/gomock"
)

func newModuleService(t *testing.T, repo modules.ModuleRepo) *modules.ModuleService {
	t.Helper()
	return modules.NewModuleService(repo)
}

func TestServiceDefaultModule(t *testing.T) {
	t.Run("batch insert error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockModuleRepo(ctrl)
		repo.EXPECT().BatchInsert(gomock.Any(), 2).Return(errors.New("insert failed"))

		svc := newModuleService(t, repo)
		if err := svc.DefaultModule(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockModuleRepo(ctrl)
		repo.EXPECT().BatchInsert(gomock.Any(), 2).DoAndReturn(
			func(mods []modules.Modules, batchSize int) error {
				if batchSize != 2 {
					t.Fatalf("expected batch size 2, got %d", batchSize)
				}
				if len(mods) != len(modules.ConstModules) {
					t.Fatalf("expected %d modules, got %d", len(modules.ConstModules), len(mods))
				}
				for i, name := range modules.ConstModules {
					if mods[i].Name != name {
						t.Fatalf("expected module %q at index %d, got %q", name, i, mods[i].Name)
					}
					if !mods[i].IsActive {
						t.Fatalf("expected module %q to be active", name)
					}
					if mods[i].ID == "" {
						t.Fatalf("expected generated id at index %d", i)
					}
				}
				return nil
			},
		)

		svc := newModuleService(t, repo)
		if err := svc.DefaultModule(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
