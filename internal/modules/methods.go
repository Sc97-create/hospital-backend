package modules

import "gorm.io/gorm"

type ModuleDb struct {
	DB *gorm.DB
}

func NewModuleDb(db *gorm.DB) *ModuleDb {
	return &ModuleDb{DB: db}
}
