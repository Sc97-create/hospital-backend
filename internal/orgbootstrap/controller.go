package orgbootstrap

import (
	"errors"
	"strings"

	"hospital-backend/pkg/logger"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Controller struct {
	Service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{Service: service}
}

func (c *Controller) AddRoles(ctx *fiber.Ctx) error {
	log := requestLogger(ctx)
	orgID, err := organisationID(ctx)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	roles, err := c.Service.AddRoles(log, orgID)
	if err != nil {
		return wrapSetupError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "roles added",
		"organisation_id": orgID,
		"roles":           roles,
	})
}

func (c *Controller) AddRolePermissions(ctx *fiber.Ctx) error {
	log := requestLogger(ctx)
	orgID, err := organisationID(ctx)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	if err := c.Service.AddRolePermissions(log, orgID); err != nil {
		return wrapSetupError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "role permissions added",
		"organisation_id": orgID,
	})
}

func (c *Controller) AddFirstUser(ctx *fiber.Ctx) error {
	log := requestLogger(ctx)
	var body struct {
		OrganisationID string `json:"organisation_id"`
		FirstName      string `json:"first_name"`
		LastName       string `json:"last_name"`
		EmailID        string `json:"email_id"`
		Username       string `json:"username"`
		MobileNumber   string `json:"mobile_number"`
		DateOfBirth    string `json:"date_of_birth"`
		DateOfJoining  string `json:"date_of_joining"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		log.Warn("organisation first user request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	result, err := c.Service.AddFirstUser(log, FirstUserRequest{
		OrganisationID: body.OrganisationID,
		FirstName:      body.FirstName,
		LastName:       body.LastName,
		EmailID:        body.EmailID,
		Username:       body.Username,
		MobileNumber:   body.MobileNumber,
		DateOfBirth:    body.DateOfBirth,
		DateOfJoining:  body.DateOfJoining,
	})
	if err != nil {
		return wrapSetupError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "first user added",
		"data":    result,
	})
}

func organisationID(ctx *fiber.Ctx) (string, error) {
	var body struct {
		OrganisationID string `json:"organisation_id"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return "", err
	}
	body.OrganisationID = strings.TrimSpace(body.OrganisationID)
	if body.OrganisationID == "" {
		return "", wrapError.ErrInvalidRequest
	}
	return body.OrganisationID, nil
}

func wrapSetupError(ctx *fiber.Ctx, err error) error {
	if errors.Is(err, wrapError.ErrInvalidRequest) {
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	}
	return wrapError.Wrap(wrapError.ErrOrganisationSetupFailed, ctx, fiber.StatusInternalServerError)
}

func requestLogger(ctx *fiber.Ctx) *zap.Logger {
	if l, ok := ctx.Locals("req_logger").(*zap.Logger); ok {
		return l
	}
	if logger.Log != nil {
		return logger.Log
	}
	return zap.NewNop()
}
