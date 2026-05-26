package rolepermissions

import (
	"errors"

	"gorm.io/gorm"
)

type RolePermissionRepo interface {
	Create(rolePermission *RolePermission) error
	BatchCreate(tx *gorm.DB, rolePermissions []RolePermission) error
	FindById(id string) (*RolePermission, error)
}

func (RPerm *RolePermissionDb) Create(rolePermission *RolePermission) error {
	return RPerm.DB.Create(rolePermission).Error
}
func (RPerm *RolePermissionDb) BatchCreate(tx *gorm.DB, rolePermissions []RolePermission) (err error) {
	tx = RPerm.DB.CreateInBatches(rolePermissions, len(rolePermissions))
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return errors.New("no role permissions inserted")
	}
	return
}
func (RPerm *RolePermissionDb) FindById(id string) (*RolePermission, error) {
	var rolePermission RolePermission
	return &rolePermission, RPerm.DB.First(&rolePermission, id).Error
}
