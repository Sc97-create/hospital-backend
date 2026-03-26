package license

import (
	"gorm.io/gorm"
)

type LicenseRepo struct {
	db *gorm.DB
}

func NewLicenseRepo(db *gorm.DB) *LicenseRepo {
	return &LicenseRepo{db: db}
}
