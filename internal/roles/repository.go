package roles

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleRepository interface {
	Create(tx *gorm.DB, role *Role) error
	BatchInsert(tx *gorm.DB, role []Role, size int) error
	FindMany(limit, offset int) ([]Role, error)
}

func (r *RoleDB) Create(tx *gorm.DB, role *Role) error {
	return tx.Create(role).Error
}
func (r *RoleDB) BatchInsert(tx *gorm.DB, role []Role, size int) error {
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).CreateInBatches(role, size).Error
}

func (r *RoleDB) FindMany(limit, offset int) ([]Role, error) {
	var roles []Role
	query := `select id,name from roles where is_default=true`
	err := r.DB.Raw(query).Limit(limit).Offset(offset).Scan(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}
