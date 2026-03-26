package rolepermissions

import "gorm.io/gorm"

type RolePermissionDb struct {
	DB *gorm.DB
}

func NewRolePermissionDb(db *gorm.DB) *RolePermissionDb {
	return &RolePermissionDb{DB: db}
}
