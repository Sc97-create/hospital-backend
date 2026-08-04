package license

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LicenseRepository interface {
	CreateLicense(log *zap.Logger, tx *gorm.DB, lic *License) (err error)
	GetLicense(log *zap.Logger, organisationID string, licensekey string) (license *License, err error)
}

func (L *LicenseRepo) CreateLicense(log *zap.Logger, tx *gorm.DB, license *License) (err error) {
	log = ensureLog(log)
	err = tx.Create(&license).Error
	if err != nil {
		log.Error("license repo error", zap.String("op", "CreateLicense"), zap.Error(err))
		return
	}
	return
}

func (L *LicenseRepo) GetLicense(log *zap.Logger, organisationID string, licensekey string) (*License, error) {
	log = ensureLog(log)
	var license License
	err := L.db.First(&license, "license_key=? and organisation_id=?", licensekey, organisationID).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("license repo error", zap.String("op", "GetLicense"), zap.Error(err))
		}
		return nil, err
	}
	return &license, nil
}
