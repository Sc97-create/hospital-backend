package organisations

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrganisationRepo interface {
	Create(log *zap.Logger, tx *gorm.DB, organisation Organisation) error
	GetByID(log *zap.Logger, query string, cond ...any) (Organisation, error)
	ListByTenantID(log *zap.Logger, query string, cond ...any) ([]Organisation, error)
	UpdateAddressByID(log *zap.Logger, query string, cond ...any) error
	Update(log *zap.Logger, query string, cond ...any) error
}

func (ORepo *OrgRepo) Create(log *zap.Logger, tx *gorm.DB, organisation Organisation) error {
	log = ensureLog(log)
	db := ORepo.db
	if tx != nil {
		db = tx
	}
	if err := db.Create(&organisation).Error; err != nil {
		log.Error("organisation repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (ORepo *OrgRepo) GetByID(log *zap.Logger, query string, cond ...any) (Organisation, error) {
	log = ensureLog(log)
	var org Organisation
	err := ORepo.db.Raw(query, cond...).Scan(&org).Error
	if err != nil {
		log.Error("organisation repo error", zap.String("op", "GetByID"), zap.Error(err))
		return Organisation{}, err
	}
	// Raw().Scan() returns nil error with zero rows — treat empty ID as not found.
	if org.ID == "" {
		return Organisation{}, gorm.ErrRecordNotFound
	}
	return org, nil
}

func (ORepo *OrgRepo) ListByTenantID(log *zap.Logger, query string, cond ...any) ([]Organisation, error) {
	log = ensureLog(log)
	var orgs []Organisation
	err := ORepo.db.Raw(query, cond...).Scan(&orgs).Error
	if err != nil {
		log.Error("organisation repo error", zap.String("op", "ListByTenantID"), zap.Error(err))
		return nil, err
	}
	return orgs, nil
}

func (ORepo *OrgRepo) UpdateAddressByID(log *zap.Logger, query string, cond ...any) error {
	log = ensureLog(log)
	result := ORepo.db.Exec(query, cond...)
	if result.Error != nil {
		log.Error("organisation repo error", zap.String("op", "UpdateAddressByID"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (ORepo *OrgRepo) Update(log *zap.Logger, query string, cond ...any) error {
	log = ensureLog(log)
	result := ORepo.db.Exec(query, cond...)
	if result.Error != nil {
		log.Error("organisation repo error", zap.String("op", "Update"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
