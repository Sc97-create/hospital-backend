package tenants

import (
	"errors"
	"strings"

	dto "hospital-backend/central/tenants/dto"
	"hospital-backend/central/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type TenantServicer interface {
	CreateTenant(log *zap.Logger, customerID string, payload dto.CreateTenantPayload) (dto.CreateTenantResult, error)
	GetTenantByID(log *zap.Logger, tenantID string) (dto.TenantDetailResult, error)
	UpdateTenantOrg(log *zap.Logger, payload dto.UpdateTenantOrgPayload) (dto.TenantDetailResult, error)
}

type TenantController struct {
	Service TenantServicer
}

type ITenantController interface {
	Create(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
}

func NewITenantController(service TenantServicer) ITenantController {
	return &TenantController{Service: service}
}

func (tc *TenantController) Create(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	customerID := middleware.GetUserID(c)
	if customerID == "" {
		logger.Warn("tenant create failed", zap.String("reason", "missing_customer"))
		return wrapError.Wrap(wrapError.ErrSessionExpired, c, fiber.StatusUnauthorized)
	}

	payload, err := bindCreatePayload(c)
	if err != nil {
		logger.Warn("tenant create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("tenant create attempt",
		zap.String("customer_id", customerID),
		zap.String("hospital_type", payload.HospitalType),
		zap.String("facility_name", payload.FacilityName),
	)

	result, err := tc.Service.CreateTenant(logger, customerID, payload)
	if err != nil {
		return wrapCreateError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "tenant created successfully",
		"tenant_id":       result.TenantID,
		"organisation_id": result.OrganisationID,
	})
}

func (tc *TenantController) GetByID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	tenantID := strings.TrimSpace(c.Params("tenant_id"))
	if tenantID == "" {
		logger.Warn("tenant get request invalid", zap.String("reason", "missing_tenant_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("tenant get attempt", zap.String("tenant_id", tenantID))
	result, err := tc.Service.GetTenantByID(logger, tenantID)
	if err != nil {
		return wrapGetError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "tenant fetched successfully",
		"data":    result,
	})
}

func (tc *TenantController) Update(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindUpdatePayload(c)
	if err != nil {
		logger.Warn("tenant update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("tenant update attempt",
		zap.String("organisation_id", payload.OrganisationID),
		zap.String("facility_name", payload.FacilityName),
	)

	result, err := tc.Service.UpdateTenantOrg(logger, payload)
	if err != nil {
		return wrapUpdateError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "tenant updated successfully",
		"data":    result,
	})
}

func bindCreatePayload(c *fiber.Ctx) (dto.CreateTenantPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.CreateTenantPayload{}, err
	}
	payload := dto.CreateTenantPayload{}
	if payload.LegalEntityName, err = requiredString(p, "legal_entity_name"); err != nil {
		return payload, err
	}
	if payload.FacilityName, err = requiredString(p, "facility_name"); err != nil {
		return payload, err
	}
	if payload.HospitalType, err = requiredString(p, "hospital_type"); err != nil {
		return payload, err
	}
	addr, err := p.GetObject("facility_address")
	if err != nil {
		return payload, wrapError.ErrInvalidRequest
	}
	if payload.FacilityAddress.Address1, err = requiredString(addr, "address1"); err != nil {
		return payload, err
	}
	payload.FacilityAddress.Address2, _ = addr.Getstring("address2")
	payload.FacilityAddress.Address2 = strings.TrimSpace(payload.FacilityAddress.Address2)
	if payload.FacilityAddress.City, err = requiredString(addr, "city"); err != nil {
		return payload, err
	}
	if payload.FacilityAddress.State, err = requiredString(addr, "state"); err != nil {
		return payload, err
	}
	return payload, nil
}

func bindUpdatePayload(c *fiber.Ctx) (dto.UpdateTenantOrgPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.UpdateTenantOrgPayload{}, err
	}
	payload := dto.UpdateTenantOrgPayload{}
	if payload.OrganisationID, err = requiredString(p, "organisation_id"); err != nil {
		return payload, err
	}
	payload.LegalEntityName, _ = p.Getstring("legal_entity_name")
	payload.FacilityName, _ = p.Getstring("facility_name")
	payload.HospitalType, _ = p.Getstring("hospital_type")
	payload.RegistrationNo, _ = p.Getstring("registration_no")
	payload.LicenseNumber, _ = p.Getstring("license_number")
	payload.GSTIN, _ = p.Getstring("gstin")
	payload.Status, _ = p.Getstring("status")
	payload.TenantStatus, _ = p.Getstring("tenant_status")
	payload.LegalEntityName = strings.TrimSpace(payload.LegalEntityName)
	payload.FacilityName = strings.TrimSpace(payload.FacilityName)
	payload.HospitalType = strings.TrimSpace(payload.HospitalType)
	payload.RegistrationNo = strings.TrimSpace(payload.RegistrationNo)
	payload.LicenseNumber = strings.TrimSpace(payload.LicenseNumber)
	payload.GSTIN = strings.TrimSpace(payload.GSTIN)
	payload.Status = strings.TrimSpace(payload.Status)
	payload.TenantStatus = strings.TrimSpace(payload.TenantStatus)

	if addr, err := p.GetObject("facility_address"); err == nil {
		payload.FacilityAddress.Address1, _ = addr.Getstring("address1")
		payload.FacilityAddress.Address2, _ = addr.Getstring("address2")
		payload.FacilityAddress.City, _ = addr.Getstring("city")
		payload.FacilityAddress.State, _ = addr.Getstring("state")
		payload.FacilityAddress.Address1 = strings.TrimSpace(payload.FacilityAddress.Address1)
		payload.FacilityAddress.Address2 = strings.TrimSpace(payload.FacilityAddress.Address2)
		payload.FacilityAddress.City = strings.TrimSpace(payload.FacilityAddress.City)
		payload.FacilityAddress.State = strings.TrimSpace(payload.FacilityAddress.State)
	}
	return payload, nil
}

func requiredString(p *params.Payload, key string) (string, error) {
	v, err := p.Getstring(key)
	if err != nil || strings.TrimSpace(v) == "" {
		return "", wrapError.ErrInvalidRequest
	}
	return strings.TrimSpace(v), nil
}

func wrapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrCustomerNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrCustomerUpdateFailed):
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	case errors.Is(err, wrapError.ErrTenantAlreadyHasOrganisation):
		return wrapError.Wrap(err, c, fiber.StatusConflict)
	case errors.Is(err, wrapError.ErrOrganisationSetupFailed):
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrTenantCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func wrapGetError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrTenantNotFound),
		errors.Is(err, wrapError.ErrOrganisationNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationFetchFailed, c, fiber.StatusInternalServerError)
	}
}

func wrapUpdateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrTenantNotFound),
		errors.Is(err, wrapError.ErrOrganisationNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrOrganisationUpdateFailed):
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrTenantUpdateFailed, c, fiber.StatusInternalServerError)
	}
}
