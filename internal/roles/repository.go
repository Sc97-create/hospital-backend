package roles

import (
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(tx *gorm.DB, role *Role) error
	InsertMany(tx *gorm.DB, role []Role) error
	FindMany(organisationID string, limit, offset int) ([]Role, error)
	Count(organisationID string) (int64, error)
	FindRoleByOrgID(organisationID string) ([]Role, error)
	FindRoleByNames(organisationID string, name string) (Role, error)
	FindByID(id string) (Role, error)
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

func (r *RoleDB) FindMany(organisationID string, limit, offset int) ([]Role, error) {
	var roles []Role
	query := `select id,name from roles where organisation_id=? LIMIT ? OFFSET ?`
	err := r.DB.Raw(query, organisationID, limit, offset).Scan(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleDB) Count(organisationID string) (int64, error) {
	var count int64
	err := r.DB.Model(&Role{}).Where("organisation_id=?", organisationID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
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

func (r *RoleDB) FindByID(id string) (Role, error) {
	var role Role
	query := `select id,name from roles where id=?`
	err := r.DB.Raw(query, id).Scan(&role).Error
	if err != nil {
		return Role{}, err
	}
	return role, nil
}
