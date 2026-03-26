package license

import (
	"hospital-backend/internal/license/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LicenseService struct {
	LicensRepo LicenseRepo
}

func NewLicenseService(repo LicenseRepo) *LicenseService {
	return &LicenseService{LicensRepo: repo}
}

func (LService *LicenseService) CreateLicenseSrv(tx *gorm.DB, orgname string, planday int, organisationID string, planspan string, issuedAt time.Time) error {
	licenseKey, expiry := utils.GenerateLicenseKey(orgname, planday, orgname, planspan, issuedAt)
	license := new(License)
	license.ID = uuid.New().String()
	license.ExpiresAt = expiry
	license.IssuedAt = time.Now()
	license.LicenseKey = licenseKey
	license.OrganisationID = organisationID
	err := LService.LicensRepo.CreateLicense(tx, license)
	if err != nil {
		return err
	}
	return nil
}
func (Lservice *LicenseService) VerifyLicense(organisationID, licensekey string) (err error) {
	lic := new(License)
	lic, err = Lservice.LicensRepo.GetLicense(organisationID, licensekey)
	if err != nil {
		return
	}
	err = utils.CompareLicenseKey(lic.LicenseKey, licensekey)
	if err != nil {
		return
	}
	return
}
