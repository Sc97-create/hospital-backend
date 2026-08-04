package organisation

import (
	"errors"
	dto "hospital-backend/internal/organisation/DTO"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type OrganisationController struct {
	Service *OrganisationService
}
type IOrganisationController interface {
	CreateOrganisation(c *fiber.Ctx) (err error)
	UpdateOrganisationLoc(c *fiber.Ctx) (err error)
	GetByID(c *fiber.Ctx) (err error)
	Update(c *fiber.Ctx) (err error)
}

func NewIOrganisationController(service *OrganisationService) IOrganisationController {
	return &OrganisationController{Service: service}
}

func (OC *OrganisationController) CreateOrganisation(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("organisation create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest := dto.OrganisationPayload{}
	payloadOrgRequest.OrganisationName, err = payload.Getstring("organisation_name")
	if err != nil || strings.TrimSpace(payloadOrgRequest.OrganisationName) == "" {
		logger.Warn("organisation create request invalid", zap.String("field", "organisation_name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.LegalEntityName, err = payload.Getstring("legal_entity_name")
	if err != nil || strings.TrimSpace(payloadOrgRequest.LegalEntityName) == "" {
		logger.Warn("organisation create request invalid", zap.String("field", "legal_entity_name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.HospitalType, err = payload.Getstring("hospital_type")
	if err != nil || strings.TrimSpace(payloadOrgRequest.HospitalType) == "" {
		logger.Warn("organisation create request invalid", zap.String("field", "hospital_type"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("organisation create attempt",
		zap.String("hospital_type", payloadOrgRequest.HospitalType),
	)

	organisationID, err := OC.Service.CreateOrganisation(logger, payloadOrgRequest)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrOrganisationCreateFailed, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "created successfully", "organisation_id": organisationID})
}

func (OC *OrganisationController) UpdateOrganisationLoc(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payloadOrgRequest := dto.OrganisationPayload{}
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("organisation location update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(payloadOrgRequest.OrganisationID) == "" {
		logger.Warn("organisation location update request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.State, err = payload.Getstring("state_id")
	if err != nil || strings.TrimSpace(payloadOrgRequest.State) == "" {
		logger.Warn("organisation location update request invalid", zap.String("field", "state_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.City, err = payload.Getstring("city_id")
	if err != nil || strings.TrimSpace(payloadOrgRequest.City) == "" {
		logger.Warn("organisation location update request invalid", zap.String("field", "city_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.Country, err = payload.Getstring("country_id")
	if err != nil || strings.TrimSpace(payloadOrgRequest.Country) == "" {
		logger.Warn("organisation location update request invalid", zap.String("field", "country_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payloadOrgRequest.AuditLogs, _ = payload.GetBool("enable_audit_logs")
	payloadOrgRequest.EmergencyAcess, _ = payload.GetBool("emergency_access")

	logger.Info("organisation location update attempt",
		zap.String("organisation_id", payloadOrgRequest.OrganisationID),
		zap.String("country_id", payloadOrgRequest.Country),
		zap.String("state_id", payloadOrgRequest.State),
		zap.String("city_id", payloadOrgRequest.City),
		zap.Bool("enable_audit_logs", payloadOrgRequest.AuditLogs),
		zap.Bool("emergency_access", payloadOrgRequest.EmergencyAcess),
	)

	err = OC.Service.UpdateOrganisationLoc(logger, payloadOrgRequest)
	if err != nil {
		return OC.wrapUpdateError(c, err)
	}
	return c.JSON(fiber.Map{"message": "updated location successfully", "code": 200})
}

func (OC *OrganisationController) GetByID(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	organisationID := strings.TrimSpace(c.Params("organisation_id"))
	if organisationID == "" {
		logger.Warn("organisation get request invalid", zap.String("reason", "missing_organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	organisation, err := OC.Service.GetOrgByID(logger, organisationID)
	if err != nil {
		return OC.wrapGetError(c, err)
	}
	return c.JSON(fiber.Map{"data": organisation, "code": 200})
}

func (OC *OrganisationController) Update(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	param, err := params.New(c)
	if err != nil {
		logger.Warn("organisation update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payload := dto.OrganisationPayload{}
	payload.OrganisationID, err = param.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(payload.OrganisationID) == "" {
		logger.Warn("organisation update request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payload.OrganisationName, err = param.Getstring("organisation_name")
	if err != nil || strings.TrimSpace(payload.OrganisationName) == "" {
		logger.Warn("organisation update request invalid", zap.String("field", "organisation_name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payload.LegalEntityName, err = param.Getstring("legal_entity_name")
	if err != nil || strings.TrimSpace(payload.LegalEntityName) == "" {
		logger.Warn("organisation update request invalid", zap.String("field", "legal_entity_name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	payload.HospitalType, err = param.Getstring("hospital_type")
	if err != nil || strings.TrimSpace(payload.HospitalType) == "" {
		logger.Warn("organisation update request invalid", zap.String("field", "hospital_type"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("organisation update attempt",
		zap.String("organisation_id", payload.OrganisationID),
		zap.String("hospital_type", payload.HospitalType),
	)

	err = OC.Service.Update(logger, payload.OrganisationID, payload)
	if err != nil {
		return OC.wrapUpdateError(c, err)
	}
	return c.JSON(fiber.Map{"message": "updated successfully", "code": 200, "organisation_id": payload.OrganisationID})
}

func (OC *OrganisationController) wrapGetError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrOrganisationNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationFetchFailed, c, fiber.StatusInternalServerError)
	}
}

func (OC *OrganisationController) wrapUpdateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrOrganisationNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationUpdateFailed, c, fiber.StatusInternalServerError)
	}
}
