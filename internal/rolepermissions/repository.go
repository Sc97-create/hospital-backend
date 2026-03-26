package rolepermissions

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RolePermissionRepo interface {
	Create(rolePermission *RolePermission) error
	BatchCreate(tx *gorm.DB, rolePermissions []RolePermission, size int) error
	FindById(id string) (*RolePermission, error)
}

func (RPerm *RolePermissionDb) Create(rolePermission *RolePermission) error {
	return RPerm.DB.Create(rolePermission).Error
}
func (RPerm *RolePermissionDb) BatchCreate(tx *gorm.DB, rolePermissions []RolePermission, size int) error {
	return RPerm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "role_id"}, {Name: "permission_id"}},
		DoNothing: true,
	}).CreateInBatches(rolePermissions, size).Error
}
func (RPerm *RolePermissionDb) FindById(id string) (*RolePermission, error) {
	var rolePermission RolePermission
	return &rolePermission, RPerm.DB.First(&rolePermission, id).Error
}
