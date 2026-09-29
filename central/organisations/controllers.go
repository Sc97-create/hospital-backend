package organisations

import (
	"errors"
	"strings"
	"time"

	dto "hospital-backend/central/organisations/dto"
	"hospital-backend/central/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type OrganisationServicer interface {
	AddOrganisation(log *zap.Logger, payload dto.OrganisationPayload) (string, error)
	UpdateAddress(log *zap.Logger, payload dto.OrganisationPayload) error
	GetOrgByID(log *zap.Logger, organisationID string) (Organisation, error)
	ListByTenantID(log *zap.Logger, tenantID string) ([]Organisation, error)
	Update(log *zap.Logger, organisationID string, payload dto.OrganisationPayload) error
}

type OrganisationController struct {
	Service OrganisationServicer
}

type IOrganisationController interface {
	AddOrganisation(c *fiber.Ctx) error
	UpdateAddress(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	ListByTenant(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
}

func NewIOrganisationController(service OrganisationServicer) IOrganisationController {
	return &OrganisationController{Service: service}
}

func (OC *OrganisationController) AddOrganisation(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindAddPayload(c)
	if err != nil {
		logger.Warn("organisation add request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("organisation add attempt",
		zap.String("tenant_id", payload.TenantID),
		zap.String("organisation_type", payload.OrganisationType),
	)

	organisationID, err := OC.Service.AddOrganisation(logger, payload)
	if err != nil {
		return wrapAddError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "organisation added successfully",
		"organisation_id": organisationID,
	})
}

func wrapAddError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrCustomerNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrOrganisationSetupFailed):
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func (OC *OrganisationController) UpdateAddress(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindAddressPayload(c)
	if err != nil {
		logger.Warn("organisation address update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("organisation address update attempt",
		zap.String("organisation_id", payload.OrganisationID),
		zap.String("country_id", payload.CountryID),
	)

	if err := OC.Service.UpdateAddress(logger, payload); err != nil {
		return wrapUpdateError(c, err)
	}
	return c.JSON(fiber.Map{"message": "updated address successfully", "code": 200})
}

func (OC *OrganisationController) GetByID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := strings.TrimSpace(c.Params("organisation_id"))
	if organisationID == "" {
		logger.Warn("organisation get request invalid", zap.String("reason", "missing_organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	organisation, err := OC.Service.GetOrgByID(logger, organisationID)
	if err != nil {
		return wrapGetError(c, err)
	}
	return c.JSON(fiber.Map{"data": organisation, "code": 200})
}

func (OC *OrganisationController) ListByTenant(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	tenantID := strings.TrimSpace(c.Params("tenant_id"))
	if tenantID == "" {
		logger.Warn("organisation list request invalid", zap.String("reason", "missing_tenant_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	orgs, err := OC.Service.ListByTenantID(logger, tenantID)
	if err != nil {
		return wrapGetError(c, err)
	}
	return c.JSON(fiber.Map{"data": orgs, "code": 200})
}

func (OC *OrganisationController) Update(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindUpdatePayload(c)
	if err != nil {
		logger.Warn("organisation update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("organisation update attempt",
		zap.String("organisation_id", payload.OrganisationID),
		zap.String("organisation_type", payload.OrganisationType),
	)

	if err := OC.Service.Update(logger, payload.OrganisationID, payload); err != nil {
		return wrapUpdateError(c, err)
	}
	return c.JSON(fiber.Map{
		"message":         "updated successfully",
		"code":            200,
		"organisation_id": payload.OrganisationID,
	})
}

func bindAddPayload(c *fiber.Ctx) (dto.OrganisationPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.OrganisationPayload{}, err
	}
	payload := dto.OrganisationPayload{}
	if payload.TenantID, err = requiredString(p, "tenant_id"); err != nil {
		return payload, err
	}
	if payload.LegalEntityName, err = requiredString(p, "legal_entity_name"); err != nil {
		return payload, err
	}
	if payload.OrganisationType, err = requiredString(p, "organisation_type"); err != nil {
		return payload, err
	}
	if payload.FacilityName, err = requiredString(p, "facility_name"); err != nil {
		return payload, err
	}
	payload.RegistrationNo, _ = p.Getstring("registration_no")
	payload.LicenseNumber, _ = p.Getstring("license_number")
	payload.GSTIN, _ = p.Getstring("gstin")
	payload.Status, _ = p.Getstring("status")
	payload.CountryID, _ = p.Getstring("country_id")
	payload.State, _ = p.Getstring("state")
	payload.City, _ = p.Getstring("city")
	bindDataSharing(&payload, p)
	expiry, err := parseOptionalDate(p, "license_expiry")
	if err != nil {
		return payload, err
	}
	payload.LicenseExpiry = expiry
	return payload, nil
}

func bindUpdatePayload(c *fiber.Ctx) (dto.OrganisationPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.OrganisationPayload{}, err
	}
	payload := dto.OrganisationPayload{}
	if payload.OrganisationID, err = requiredString(p, "organisation_id"); err != nil {
		return payload, err
	}
	payload.LegalEntityName, _ = p.Getstring("legal_entity_name")
	payload.OrganisationType, _ = p.Getstring("organisation_type")
	payload.FacilityName, _ = p.Getstring("facility_name")
	payload.RegistrationNo, _ = p.Getstring("registration_no")
	payload.LicenseNumber, _ = p.Getstring("license_number")
	payload.GSTIN, _ = p.Getstring("gstin")
	payload.Status, _ = p.Getstring("status")
	expiry, err := parseOptionalDate(p, "license_expiry")
	if err != nil {
		return payload, err
	}
	payload.LicenseExpiry = expiry
	return payload, nil
}

func bindAddressPayload(c *fiber.Ctx) (dto.OrganisationPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.OrganisationPayload{}, err
	}
	payload := dto.OrganisationPayload{}
	if payload.OrganisationID, err = requiredString(p, "organisation_id"); err != nil {
		return payload, err
	}
	if payload.CountryID, err = requiredString(p, "country_id"); err != nil {
		return payload, err
	}
	if payload.State, err = requiredString(p, "state_id"); err != nil {
		return payload, err
	}
	if payload.City, err = requiredString(p, "city_id"); err != nil {
		return payload, err
	}
	bindDataSharing(&payload, p)
	return payload, nil
}

func bindDataSharing(payload *dto.OrganisationPayload, p *params.Payload) {
	payload.PatientLookup, _ = p.GetBool("patient_lookup")
	payload.LabReports, _ = p.GetBool("lab_reports")
}

func requiredString(p *params.Payload, key string) (string, error) {
	v, err := p.Getstring(key)
	if err != nil || strings.TrimSpace(v) == "" {
		return "", wrapError.ErrInvalidRequest
	}
	return strings.TrimSpace(v), nil
}

func parseOptionalDate(p *params.Payload, key string) (*time.Time, error) {
	raw, err := p.Getstring(key)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return nil, wrapError.ErrInvalidRequest
	}
	return &parsed, nil
}

func wrapGetError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrOrganisationNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationFetchFailed, c, fiber.StatusInternalServerError)
	}
}

func wrapUpdateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrOrganisationNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationUpdateFailed, c, fiber.StatusInternalServerError)
	}
}
