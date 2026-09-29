package modules

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ModuleService struct {
	repo ModuleRepo
}

func NewModuleService(repo ModuleRepo) *ModuleService {
	return &ModuleService{repo: repo}
}

func (Mod *ModuleService) DefaultModule(log *zap.Logger) error {
	log = ensureLog(log)
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
	err := Mod.repo.BatchInsert(log, moduleArr, 2)
	if err != nil {
		log.Error("module seed failed", zap.String("reason", "db_insert"), zap.Error(err))
		return err
	}
	log.Info("module seed success", zap.Int("count", len(moduleArr)))
	return nil
}
