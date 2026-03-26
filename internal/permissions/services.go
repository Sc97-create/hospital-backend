package permissions

import (
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions/dto"
	"hospital-backend/internal/permissions/utils"
	"time"

	"github.com/google/uuid"
)

type PermService struct {
	PermissionRepo PermissionRepo
	ModuleDb       *modules.ModuleDb
}

func NewService(PermRepo PermissionRepo, ModuleDb *modules.ModuleDb) *PermService {
	return &PermService{PermissionRepo: PermRepo, ModuleDb: ModuleDb}
}
func (PermSer *PermService) DefaultPerm() error {
	now := time.Now()
	permArr := []Permission{}
	for _, name := range utils.AdminPermArr {
		permArr = append(permArr, Permission{
			ID:        uuid.NewString(),
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	err := PermSer.PermissionRepo.BatchInsert(permArr, 2)
	if err != nil {
		return err
	}
	return nil
}
func (PermSer *PermService) FindMany() ([]dto.ModuleResponse, []dto.PermissionResponse, error) {
	permissions, err := PermSer.PermissionRepo.FindMany()
	if err != nil {
		return nil, nil, err
	}
	modules, err := PermSer.ModuleDb.FindMany()
	if err != nil {
		return nil, nil, err
	}
	moduleResp := []dto.ModuleResponse{}
	permissionResp := []dto.PermissionResponse{}
	for _, each := range modules {
		moduleResp = append(moduleResp, dto.ModuleResponse{
			ID:   each.ID,
			Name: each.Name,
		})
	}
	for _, each := range permissions {
		permissionResp = append(permissionResp, dto.PermissionResponse{
			ID:   each.ID,
			Name: each.Name,
		})
	}
	return moduleResp, permissionResp, nil
}
