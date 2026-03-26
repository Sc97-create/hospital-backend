package license

import "gorm.io/gorm"

type LicenseRepository interface {
	CreateLicense(tx *gorm.DB, lic *License) (err error)
	GetLicense(licenseID string) (license *License, err error)
}

func (L *LicenseRepo) CreateLicense(tx *gorm.DB, license *License) (err error) {
	err = tx.Create(&license).Error
	if err != nil {
		return
	}
	return
}
func (L *LicenseRepo) GetLicense(organisationID string, licensekey string) (*License, error) {
	var license License
	err := L.db.First(&license, "license_key=? and organisation_id=?", licensekey, organisationID).Error
	if err != nil {
		return nil, err
	}
	return &license, nil
}
