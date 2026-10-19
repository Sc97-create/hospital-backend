package labservice

import (
	"errors"
	"strings"

	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type LabTestTemplateServicer interface {
	ListDefaults(log *zap.Logger, organisationID string) (dto.DefaultsListResponse, error)
	Create(log *zap.Logger, req dto.CreateLabTestTemplateRequest) (string, error)
	GetByID(log *zap.Logger, id string) (dto.LabTestTemplateResponse, error)
	List(log *zap.Logger, req dto.ListLabTestTemplatesRequest) ([]dto.LabTestTemplateResponse, int64, error)
}

type LabTestTemplateController struct {
	Service LabTestTemplateServicer
}

func NewLabTestTemplateController(service LabTestTemplateServicer) *LabTestTemplateController {
	return &LabTestTemplateController{Service: service}
}

func (c *LabTestTemplateController) ListDefaults(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	organisationID := strings.TrimSpace(ctx.Query("organisation_id"))
	if organisationID == "" {
		logger.Warn("lab test defaults request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	data, err := c.Service.ListDefaults(logger, organisationID)
	if err != nil {
		return c.wrapReadError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":            200,
		"defaults":        data.Defaults,
		"templates":       data.Templates,
		"defaults_count":  len(data.Defaults),
		"templates_count": len(data.Templates),
	})
}

func (c *LabTestTemplateController) Create(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("lab test template create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	req, err := c.parseCreateTemplateRequest(payload)
	if err != nil {
		logger.Warn("lab test template create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	id, err := c.Service.Create(logger, req)
	if err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":                 200,
		"message":              "lab test template created successfully",
		"lab_test_template_id": id,
	})
}

func (c *LabTestTemplateController) parseCreateTemplateRequest(payload *params.Payload) (dto.CreateLabTestTemplateRequest, error) {
	var req dto.CreateLabTestTemplateRequest
	var err error
	req.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(req.OrganisationID) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.CreatedBy, err = payload.Getstring("created_by")
	if err != nil || strings.TrimSpace(req.CreatedBy) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.Name, err = payload.Getstring("name")
	if err != nil || strings.TrimSpace(req.Name) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.Description, _ = payload.Getstring("description")

	if codes, codeErr := payload.GetStringArray("test_codes"); codeErr == nil && len(codes) > 0 {
		req.TestCodes = codes
		return req, nil
	}

	children, childErr := payload.GetChildren("tests")
	if childErr != nil || len(children) == 0 {
		return req, wrapError.ErrInvalidRequest
	}
	req.Tests = make([]dto.LabTemplatePayload, 0, len(children))
	for _, child := range children {
		code, codeErr := child.Getstring("test_code")
		name, nameErr := child.Getstring("test_name")
		if codeErr != nil || nameErr != nil || strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" {
			return req, wrapError.ErrInvalidRequest
		}
		testType, _ := child.Getstring("type")
		description, _ := child.Getstring("description")
		tatMinutes, _ := child.Getint("tat_minutes")
		specimens, _ := child.GetStringArray("specimens")
		req.Tests = append(req.Tests, dto.LabTemplatePayload{
			TestCode:    code,
			TestName:    name,
			Specimens:   specimens,
			Type:        testType,
			TATMinutes:  tatMinutes,
			Description: description,
		})
	}
	return req, nil
}

func (c *LabTestTemplateController) List(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	var req dto.ListLabTestTemplatesRequest
	if err := ctx.QueryParser(&req); err != nil {
		logger.Warn("lab test template list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	if strings.TrimSpace(req.OrganisationID) == "" {
		logger.Warn("lab test template list request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	rows, total, err := c.Service.List(logger, req)
	if err != nil {
		return c.wrapReadError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":  200,
		"data":  rows,
		"count": total,
	})
}

func (c *LabTestTemplateController) GetByID(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		logger.Warn("lab test template get request invalid", zap.String("field", "id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	row, err := c.Service.GetByID(logger, id)
	if err != nil {
		return c.wrapReadError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code": 200,
		"data": row,
	})
}

func (c *LabTestTemplateController) wrapReadError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrLabTestTemplateNotFound):
		return wrapError.Wrap(err, ctx, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrLabTestTemplateFetchFailed, ctx, fiber.StatusInternalServerError)
	}
}

func (c *LabTestTemplateController) wrapWriteError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrLabTestTemplateCreateFailed):
		return wrapError.Wrap(err, ctx, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrLabTestTemplateCreateFailed, ctx, fiber.StatusInternalServerError)
	}
}
