package diagnosticcentre

import (
	"errors"
	"strings"

	"hospital-backend/internal/labService/diagnosticCentre/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type DiagnosticCentreServicer interface {
	Create(log *zap.Logger, req dto.CreateDiagnosticCentreRequest) (string, error)
	List(log *zap.Logger, req dto.ListDiagnosticCentresRequest) ([]dto.DiagnosticCentreResponse, int64, error)
	Update(log *zap.Logger, id string, req dto.UpdateDiagnosticCentreRequest) error
	UpdateStatus(log *zap.Logger, id, status string) error
}

type DiagnosticCentreController struct {
	Service DiagnosticCentreServicer
}

func NewDiagnosticCentreController(service DiagnosticCentreServicer) *DiagnosticCentreController {
	return &DiagnosticCentreController{Service: service}
}

func (c *DiagnosticCentreController) Create(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("diagnostic centre create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	var req dto.CreateDiagnosticCentreRequest
	req.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(req.OrganisationID) == "" {
		logger.Warn("diagnostic centre create request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	req.Name, err = payload.Getstring("name")
	if err != nil || strings.TrimSpace(req.Name) == "" {
		logger.Warn("diagnostic centre create request invalid", zap.String("field", "name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	req.Address, _ = payload.Getstring("address")
	req.ContactNumber, _ = payload.Getstring("contact_number")
	req.Email, _ = payload.Getstring("email")

	id, err := c.Service.Create(logger, req)
	if err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":                 200,
		"message":              "diagnostic centre created successfully",
		"diagnostic_centre_id": id,
	})
}

func (c *DiagnosticCentreController) List(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	var req dto.ListDiagnosticCentresRequest
	if err := ctx.QueryParser(&req); err != nil {
		logger.Warn("diagnostic centre list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	if strings.TrimSpace(req.OrganisationID) == "" {
		logger.Warn("diagnostic centre list request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	rows, total, err := c.Service.List(logger, req)
	if err != nil {
		return c.wrapReadError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":  200,
		"data":  rows,
		"total": total,
	})
}

func (c *DiagnosticCentreController) Update(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		logger.Warn("diagnostic centre update request invalid", zap.String("field", "id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("diagnostic centre update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	var req dto.UpdateDiagnosticCentreRequest
	req.Name, err = payload.Getstring("name")
	if err != nil || strings.TrimSpace(req.Name) == "" {
		logger.Warn("diagnostic centre update request invalid", zap.String("field", "name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	req.Address, _ = payload.Getstring("address")
	req.ContactNumber, _ = payload.Getstring("contact_number")
	req.Email, _ = payload.Getstring("email")

	if err := c.Service.Update(logger, id, req); err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":    200,
		"message": "diagnostic centre updated successfully",
	})
}

func (c *DiagnosticCentreController) UpdateStatus(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		logger.Warn("diagnostic centre status request invalid", zap.String("field", "id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("diagnostic centre status request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	status, err := payload.Getstring("status")
	if err != nil || strings.TrimSpace(status) == "" {
		logger.Warn("diagnostic centre status request invalid", zap.String("field", "status"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	if err := c.Service.UpdateStatus(logger, id, status); err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":    200,
		"message": "diagnostic centre status updated successfully",
	})
}

func (c *DiagnosticCentreController) wrapReadError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrDiagnosticCentreNotFound):
		return wrapError.Wrap(err, ctx, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrDiagnosticCentreFetchFailed, ctx, fiber.StatusInternalServerError)
	}
}

func (c *DiagnosticCentreController) wrapWriteError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrDiagnosticCentreNotFound):
		return wrapError.Wrap(err, ctx, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrDiagnosticCentreCreateFailed):
		return wrapError.Wrap(err, ctx, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrDiagnosticCentreUpdateFailed, ctx, fiber.StatusInternalServerError)
	}
}
