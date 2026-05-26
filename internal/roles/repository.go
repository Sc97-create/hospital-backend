package roles

import (
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(tx *gorm.DB, role *Role) error
	InsertMany(tx *gorm.DB, role []Role) error
	FindMany(limit, offset int) ([]Role, error)
	FindRoleByOrgID(organisationID string) ([]Role, error)
	FindRoleByNames(organisationID string, name string) (Role, error)
}

func (r *RoleDB) Create(tx *gorm.DB, role *Role) error {
	return tx.Create(role).Error
}
func (r *RoleDB) InsertMany(tx *gorm.DB, role []Role) (err error) {
	err = tx.CreateInBatches(role, len(role)).Error
	if err != nil {
		return
	}
	return
}

func (r *RoleDB) FindMany(limit, offset int) ([]Role, error) {
	var roles []Role
	query := `select id,name from roles where is_default=true LIMIT ? OFFSET ?`
	err := r.DB.Raw(query, limit, offset).Scan(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}
func (r *RoleDB) FindRoleByOrgID(organisationID string) ([]Role, error) {
	var roles []Role
	query := `select id,name from roles where organisation_id=?`
	err := r.DB.Model(&Role{}).Raw(query, organisationID).Scan(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}
func (r *RoleDB) FindRoleByNames(organisationID string, name string) (Role, error) {
	var role Role
	query := `select id,name from roles where organisation_id=? and name=?`
	err := r.DB.Raw(query, organisationID, name).Scan(&role).Error
	if err != nil {
		return Role{}, err
	}
	return role, nil
}
