package license

import (
	"errors"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func VerifyLicense(c *fiber.Ctx, service *LicenseService) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("license verify request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	organisationID := strings.TrimSpace(c.Params("organisationID"))
	if organisationID == "" {
		logger.Warn("license verify request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	licensekey, err := payload.Getstring("license_key")
	if err != nil || strings.TrimSpace(licensekey) == "" {
		logger.Warn("license verify request invalid", zap.String("field", "license_key"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("license verify attempt", zap.String("organisation_id", organisationID))

	err = service.VerifyLicense(logger, organisationID, licensekey)
	if err != nil {
		return wrapVerifyError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "license verified", "code": "200"})
}

func wrapVerifyError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrLicenseNotFound),
		errors.Is(err, wrapError.ErrLicenseInvalid):
		return wrapError.Wrap(err, c, fiber.StatusUnauthorized)
	default:
		return wrapError.Wrap(wrapError.ErrLicenseVerifyFailed, c, fiber.StatusInternalServerError)
	}
}
