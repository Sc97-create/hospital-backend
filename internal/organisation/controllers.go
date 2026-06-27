package organisation

import (
	dto "hospital-backend/internal/organisation/DTO"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
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
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest := dto.OrganisationPayload{}
	payloadOrgRequest.OrganisationName, err = payload.Getstring("organisation_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest.LegalEntityName, err = payload.Getstring("legal_entity_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	payloadOrgRequest.HospitalType, err = payload.Getstring("hospital_type")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	organisationID, err := OC.Service.CreateOrganisation(payloadOrgRequest)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "created successfully", "organisation_id": organisationID})
}
func (OC *OrganisationController) UpdateOrganisationLoc(c *fiber.Ctx) (err error) {
	payloadOrgRequest := dto.OrganisationPayload{}
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest.State, err = payload.Getstring("state_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest.City, err = payload.Getstring("city_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest.Country, err = payload.Getstring("country_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	payloadOrgRequest.AuditLogs, _ = payload.GetBool("enable_audit_logs")
	payloadOrgRequest.EmergencyAcess, _ = payload.GetBool("emergency_access")
	err = OC.Service.UpdateOrganisationLoc(payloadOrgRequest)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"message": "updated location successfully", "code": 200})
}
func (OC *OrganisationController) GetByID(c *fiber.Ctx) (err error) {
	organisationID := c.Params("organisation_id")
	organisation, err := OC.Service.GetByID(organisationID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"data": organisation, "code": 200})
}
func (OC *OrganisationController) Update(c *fiber.Ctx) (err error) {
	param, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	dto := dto.OrganisationPayload{}
	dto.OrganisationID, err = param.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	dto.OrganisationName, err = param.Getstring("organisation_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	dto.LegalEntityName, err = param.Getstring("legal_entity_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	dto.HospitalType, err = param.Getstring("hospital_type")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	err = OC.Service.Update(dto.OrganisationID, dto)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"message": "updated successfully", "code": 200, "organisation_id": dto.OrganisationID})

}
