package modules

import (
	"time"

	"github.com/google/uuid"
)

type ModuleService struct {
	ModuleDb *ModuleDb
}

func NewModuleService(ModuleDb *ModuleDb) *ModuleService {
	return &ModuleService{ModuleDb: ModuleDb}
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
	err := Mod.ModuleDb.BatchInsert(moduleArr, 2)
	if err != nil {
		return err
	}
	return nil
}
