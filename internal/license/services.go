package license

import (
	"errors"
	"hospital-backend/internal/license/utils"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LicenseService struct {
	LicenseRepo LicenseRepository
}

func NewLicenseService(repo LicenseRepository) *LicenseService {
	return &LicenseService{LicenseRepo: repo}
}

func (LService *LicenseService) CreateLicenseSrv(log *zap.Logger, tx *gorm.DB, orgname string, planday int, organisationID string, planspan string, issuedAt time.Time) error {
	log = ensureLog(log)
	licenseKey, expiry := utils.GenerateLicenseKey(orgname, planday, orgname, planspan, issuedAt)
	license := new(License)
	license.ID = uuid.New().String()
	license.ExpiresAt = expiry
	license.IssuedAt = time.Now()
	license.LicenseKey = licenseKey
	license.OrganisationID = organisationID
	err := LService.LicenseRepo.CreateLicense(log, tx, license)
	if err != nil {
		log.Error("license create failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return wrapError.ErrLicenseCreateFailed
	}
	log.Info("license create success",
		zap.String("organisation_id", organisationID),
		zap.String("license_id", license.ID),
		zap.String("planspan", planspan),
		zap.Int("planday", planday),
	)
	return nil
}

func (Lservice *LicenseService) VerifyLicense(log *zap.Logger, organisationID, licensekey string) error {
	log = ensureLog(log)
	organisationID = strings.TrimSpace(organisationID)
	licensekey = strings.TrimSpace(licensekey)
	if organisationID == "" || licensekey == "" {
		log.Warn("license verify failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "invalid_request"),
		)
		return wrapError.ErrInvalidRequest
	}

	lic, err := Lservice.LicenseRepo.GetLicense(log, organisationID, licensekey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("license verify failed",
				zap.String("organisation_id", organisationID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrLicenseNotFound
		}
		log.Error("license verify failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return wrapError.ErrLicenseVerifyFailed
	}

	err = utils.CompareLicenseKey(lic.LicenseKey, licensekey)
	if err != nil {
		reason := "invalid_key"
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "format") {
			reason = "invalid_format"
		} else if strings.Contains(msg, "tampered") || strings.Contains(msg, "expiry") {
			reason = "expiry_mismatch"
		}
		log.Warn("license verify failed",
			zap.String("organisation_id", organisationID),
			zap.String("license_id", lic.ID),
			zap.String("reason", reason),
		)
		return wrapError.ErrLicenseInvalid
	}

	log.Info("license verify success",
		zap.String("organisation_id", organisationID),
		zap.String("license_id", lic.ID),
	)
	return nil
}
