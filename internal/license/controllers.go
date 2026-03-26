package license

import (
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

func VerifyLicense(c *fiber.Ctx, service *LicenseService) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	organisationID := c.Params("organisation_id")
	licensekey, err := payload.Getstring("license_key")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	err = service.VerifyLicense(organisationID, licensekey)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "license verified", "code": "200"})
}
