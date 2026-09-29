package roles

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(log *zap.Logger, tx *gorm.DB, role *Role) error
	InsertMany(log *zap.Logger, tx *gorm.DB, role []Role) error
	FindMany(log *zap.Logger, organisationID string, limit, offset int) ([]Role, error)
	Count(log *zap.Logger, organisationID string) (int64, error)
	FindRoleByOrgID(log *zap.Logger, organisationID string) ([]Role, error)
	FindRoleByNames(log *zap.Logger, organisationID string, name string) (Role, error)
	FindByID(log *zap.Logger, id string) (Role, error)
}

func (r *RoleDB) Create(log *zap.Logger, tx *gorm.DB, role *Role) error {
	log = ensureLog(log)
	err := tx.Create(role).Error
	if err != nil {
		logDBError(log, "Create", err)
	}
	return err
}

func (r *RoleDB) InsertMany(log *zap.Logger, tx *gorm.DB, role []Role) (err error) {
	log = ensureLog(log)
	err = tx.CreateInBatches(role, len(role)).Error
	if err != nil {
		logDBError(log, "InsertMany", err)
		return
	}
	return
}

func (r *RoleDB) FindMany(log *zap.Logger, organisationID string, limit, offset int) ([]Role, error) {
	log = ensureLog(log)
	var roles []Role
	query := `select id,name from roles where organisation_id=? LIMIT ? OFFSET ?`
	err := r.DB.Raw(query, organisationID, limit, offset).Scan(&roles).Error
	if err != nil {
		logDBError(log, "FindMany", err)
		return nil, err
	}
	return roles, nil
}

func (r *RoleDB) Count(log *zap.Logger, organisationID string) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := r.DB.Model(&Role{}).Where("organisation_id=?", organisationID).Count(&count).Error
	if err != nil {
		logDBError(log, "Count", err)
		return 0, err
	}
	return count, nil
}

func (r *RoleDB) FindRoleByOrgID(log *zap.Logger, organisationID string) ([]Role, error) {
	log = ensureLog(log)
	var roles []Role
	query := `select id,name from roles where organisation_id=?`
	err := r.DB.Model(&Role{}).Raw(query, organisationID).Scan(&roles).Error
	if err != nil {
		logDBError(log, "FindRoleByOrgID", err)
		return nil, err
	}
	return roles, nil
}

func (r *RoleDB) FindRoleByNames(log *zap.Logger, organisationID string, name string) (Role, error) {
	log = ensureLog(log)
	var role Role
	query := `select id,name from roles where organisation_id=? and name=?`
	err := r.DB.Raw(query, organisationID, name).Scan(&role).Error
	if err != nil {
		logDBError(log, "FindRoleByNames", err)
		return Role{}, err
	}
	return role, nil
}

func (r *RoleDB) FindByID(log *zap.Logger, id string) (Role, error) {
	log = ensureLog(log)
	var role Role
	query := `select id,name from roles where id=?`
	err := r.DB.Raw(query, id).Scan(&role).Error
	if err != nil {
		logDBError(log, "FindByID", err)
		return Role{}, err
	}
	return role, nil
}
