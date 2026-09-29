package modules

import (
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

type ModuleRepo interface {
	BatchInsert(log *zap.Logger, modules []Modules, size int) error
	FindMany(log *zap.Logger) ([]Modules, error)
}

func (Mod *ModuleDb) BatchInsert(log *zap.Logger, modules []Modules, size int) error {
	log = ensureLog(log)
	err := Mod.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).CreateInBatches(modules, size).Error
	if err != nil {
		logDBError(log, "BatchInsert", err)
	}
	return err
}

func (Mod *ModuleDb) FindMany(log *zap.Logger) ([]Modules, error) {
	log = ensureLog(log)
	query := `select id,name from modules where is_active=true`
	var modules []Modules
	err := Mod.DB.Raw(query).Scan(&modules).Error
	if err != nil {
		logDBError(log, "FindMany", err)
		return nil, err
	}
	return modules, nil
}
