package modules

import (
	"gorm.io/gorm/clause"
)

type ModuleRepo interface {
	BatchInsert([]Modules, int) error
	FindMany() ([]Modules, error)
}

func (Mod *ModuleDb) BatchInsert(modules []Modules, size int) error {
	return Mod.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).CreateInBatches(modules, size).Error
}
func (Mod *ModuleDb) FindMany() ([]Modules, error) {
	query := `select id,name from modules where is_active=true`
	var modules []Modules
	err := Mod.DB.Raw(query).Scan(&modules).Error
	if err != nil {
		return nil, err
	}
	return modules, nil
}
