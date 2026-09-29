package permissions

import (
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

type PermissionRepo interface {
	BatchInsert(log *zap.Logger, permissions []Permission, size int) error
	FindMany(log *zap.Logger) ([]Permission, error)
	GetPermissionByName(log *zap.Logger) ([]string, error)
}

func (Perm *PermissionDB) BatchInsert(log *zap.Logger, permission []Permission, size int) error {
	log = ensureLog(log)
	err := Perm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).CreateInBatches(permission, size).Error
	if err != nil {
		logDBError(log, "BatchInsert", err)
	}
	return err
}

func (Perm *PermissionDB) GetPermissionByName(log *zap.Logger) ([]string, error) {
	log = ensureLog(log)
	query := `select id from permissions where name in ('create','update','delete','view')`
	var permission []string
	err := Perm.DB.Raw(query).Scan(&permission).Error
	if err != nil {
		logDBError(log, "GetPermissionByName", err)
		return nil, err
	}
	return permission, nil
}

func (Perm *PermissionDB) FindMany(log *zap.Logger) ([]Permission, error) {
	log = ensureLog(log)
	query := `select id,name from permissions`
	var permissions []Permission
	err := Perm.DB.Raw(query).Scan(&permissions).Error
	if err != nil {
		logDBError(log, "FindMany", err)
		return nil, err
	}
	return permissions, nil
}
