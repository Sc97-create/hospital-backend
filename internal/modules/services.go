package modules

import (
	"time"

	"github.com/google/uuid"
)

type ModuleService struct {
	repo ModuleRepo
}

func NewModuleService(repo ModuleRepo) *ModuleService {
	return &ModuleService{repo: repo}
}

func (Mod *ModuleService) DefaultModule() error {
	moduleArr := []Modules{}
	for _, each := range ConstModules {
		moduleArr = append(moduleArr, Modules{
			ID:        uuid.New().String(),
			Name:      each,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		})
	}
	err := Mod.repo.BatchInsert(moduleArr, 2)
	if err != nil {
		return err
	}
	return nil
}
