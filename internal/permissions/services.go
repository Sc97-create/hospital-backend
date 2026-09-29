package permissions

import (
	"hospital-backend/internal/modules"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PermService struct {
	PermissionRepo PermissionRepo
	ModuleLookup   modules.ModuleRepo
}

func NewService(PermRepo PermissionRepo, moduleLookup modules.ModuleRepo) *PermService {
	return &PermService{PermissionRepo: PermRepo, ModuleLookup: moduleLookup}
}

func (PermSer *PermService) DefaultPerm(log *zap.Logger) error {
	log = ensureLog(log)
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
	if err := PermSer.PermissionRepo.BatchInsert(log, permArr, 2); err != nil {
		log.Error("permission seed failed", zap.String("reason", "db_insert"), zap.Error(err))
		return err
	}
	log.Info("permission seed success", zap.Int("count", len(permArr)))
	return nil
}

func (PermSer *PermService) FindMany(log *zap.Logger) ([]modules.Modules, []Permission, error) {
	log = ensureLog(log)
	permissions, err := PermSer.PermissionRepo.FindMany(log)
	if err != nil {
		log.Error("permission list failed", zap.String("reason", "db_permissions"), zap.Error(err))
		return nil, nil, wrapError.ErrPermissionsFetchFailed
	}
	modules, err := PermSer.ModuleLookup.FindMany(log)
	if err != nil {
		log.Error("permission list failed", zap.String("reason", "db_modules"), zap.Error(err))
		return nil, nil, wrapError.ErrPermissionsFetchFailed
	}
	return modules, permissions, nil
}
