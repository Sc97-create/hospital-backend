package permissions

import "gorm.io/gorm"

type PermissionDB struct {
	DB *gorm.DB
}

func NewPermDB(db *gorm.DB) *PermissionDB {
	return &PermissionDB{DB: db}
}
