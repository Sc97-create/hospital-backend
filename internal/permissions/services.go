package permissions

import (
	"hospital-backend/internal/modules"
	"time"

	"github.com/google/uuid"
)

type PermService struct {
	PermissionRepo PermissionRepo
	ModuleLookup   modules.ModuleRepo
}

func NewService(PermRepo PermissionRepo, moduleLookup modules.ModuleRepo) *PermService {
	return &PermService{PermissionRepo: PermRepo, ModuleLookup: moduleLookup}
}

func (PermSer *PermService) DefaultPerm() error {
	now := time.Now()
	permArr := []Permission{}
	for _, name := range AdminPermArr {
		permArr = append(permArr, Permission{
			ID:        uuid.NewString(),
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return PermSer.PermissionRepo.BatchInsert(permArr, 2)
}

func (PermSer *PermService) FindMany() ([]modules.Modules, []Permission, error) {
	permissions, err := PermSer.PermissionRepo.FindMany()
	if err != nil {
		return nil, nil, err
	}
	modules, err := PermSer.ModuleLookup.FindMany()
	if err != nil {
		return nil, nil, err
	}
	return modules, permissions, nil
}
