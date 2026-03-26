package permissions

import (
	"gorm.io/gorm/clause"
)

type PermissionRepo interface {
	BatchInsert([]Permission, int) error
	FindMany() ([]Permission, error)
	GetPermissionByName() ([]string, error)
}

func (Perm *PermissionDB) BatchInsert(permission []Permission, size int) error {
	return Perm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).CreateInBatches(permission, size).Error
}
func (Perm *PermissionDB) GetPermissionByName() ([]string, error) {
	query := `select id from permissions where name in ('create','update','delete','view')`
	var permission []string
	err := Perm.DB.Raw(query).Scan(&permission).Error
	if err != nil {
		return nil, err
	}
	return permission, nil
}
func (Perm *PermissionDB) FindMany() ([]Permission, error) {
	query := `select id,name from permissions`
	var permissions []Permission
	err := Perm.DB.Raw(query).Scan(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}
