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

type LabOrderServicer interface {
	Create(log *zap.Logger, req dto.CreateLabOrderRequest) (string, error)
	GetByID(log *zap.Logger, id string) (dto.LabOrderResponse, error)
	List(log *zap.Logger, req dto.ListLabOrdersRequest) ([]dto.LabOrderListItem, int64, error)
	UpdateStatus(log *zap.Logger, id, status string) error
	UpdateTestStatus(log *zap.Logger, orderID, testID, status string) error
	UpdateTestsStatus(log *zap.Logger, orderID string, testIDs []string, status string) error
}

type LabOrderController struct {
	Service LabOrderServicer
}

func NewLabOrderController(service LabOrderServicer) *LabOrderController {
	return &LabOrderController{Service: service}
}

func (c *LabOrderController) Create(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("lab order create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	req, err := c.parseCreateRequest(payload)
	if err != nil {
		logger.Warn("lab order create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	id, err := c.Service.Create(logger, req)
	if err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":         200,
		"message":      "lab order created successfully",
		"lab_order_id": id,
	})
}

func (c *LabOrderController) parseCreateRequest(payload *params.Payload) (dto.CreateLabOrderRequest, error) {
	var req dto.CreateLabOrderRequest
	var err error
	req.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(req.OrganisationID) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.PatientID, err = payload.Getstring("patient_id")
	if err != nil || strings.TrimSpace(req.PatientID) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.DoctorID, err = payload.Getstring("doctor_id")
	if err != nil || strings.TrimSpace(req.DoctorID) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.DiagnosticCentreID, err = payload.Getstring("diagnostic_centre_id")
	if err != nil || strings.TrimSpace(req.DiagnosticCentreID) == "" {
		return req, wrapError.ErrInvalidRequest
	}
	req.AppointmentID, _ = payload.Getstring("appointment_id")

	children, err := payload.GetChildren("tests")
	if err != nil || len(children) == 0 {
		return req, wrapError.ErrInvalidRequest
	}
	req.Tests = make([]dto.CreateOrderedTestInput, 0, len(children))
	for _, child := range children {
		name, nameErr := child.Getstring("name")
		if nameErr != nil || strings.TrimSpace(name) == "" {
			return req, wrapError.ErrInvalidRequest
		}
		subTest, _ := child.Getstring("sub_test")
		req.Tests = append(req.Tests, dto.CreateOrderedTestInput{
			Name:    name,
			SubTest: subTest,
		})
	}
	return req, nil
}

func (c *LabOrderController) List(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	var req dto.ListLabOrdersRequest
	if err := ctx.QueryParser(&req); err != nil {
		logger.Warn("lab order list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	if strings.TrimSpace(req.OrganisationID) == "" {
		logger.Warn("lab order list request invalid", zap.String("field", "organisation_id"))
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

func (c *LabOrderController) GetByID(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		logger.Warn("lab order get request invalid", zap.String("field", "id"))
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

func (c *LabOrderController) UpdateStatus(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	id := strings.TrimSpace(ctx.Params("id"))
	if id == "" {
		logger.Warn("lab order status request invalid", zap.String("field", "id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("lab order status request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	status, err := payload.Getstring("status")
	if err != nil || strings.TrimSpace(status) == "" {
		logger.Warn("lab order status request invalid", zap.String("field", "status"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	if err := c.Service.UpdateStatus(logger, id, status); err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":    200,
		"message": "lab order status updated successfully",
	})
}

func (c *LabOrderController) UpdateTestStatus(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	orderID := strings.TrimSpace(ctx.Params("id"))
	testID := strings.TrimSpace(ctx.Params("testId"))
	if orderID == "" || testID == "" {
		logger.Warn("lab order test status request invalid", zap.String("field", "id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("lab order test status request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	status, err := payload.Getstring("status")
	if err != nil || strings.TrimSpace(status) == "" {
		logger.Warn("lab order test status request invalid", zap.String("field", "status"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	if err := c.Service.UpdateTestStatus(logger, orderID, testID, status); err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":    200,
		"message": "lab order test status updated successfully",
	})
}

func (c *LabOrderController) UpdateTestsStatus(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	payload, err := params.New(ctx)
	if err != nil {
		logger.Warn("lab order bulk status request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	orderID, err := payload.Getstring("lab_order_id")
	if err != nil || strings.TrimSpace(orderID) == "" {
		logger.Warn("lab order bulk status request invalid", zap.String("field", "lab_order_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	status, err := payload.Getstring("status")
	if err != nil || strings.TrimSpace(status) == "" {
		logger.Warn("lab order bulk status request invalid", zap.String("field", "status"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	testIDs, err := payload.GetStringArray("test_ids")
	if err != nil || len(testIDs) == 0 {
		logger.Warn("lab order bulk status request invalid", zap.String("field", "test_ids"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}

	if err := c.Service.UpdateTestsStatus(logger, orderID, testIDs, status); err != nil {
		return c.wrapWriteError(ctx, err)
	}
	return ctx.JSON(fiber.Map{
		"code":    200,
		"message": "lab order test statuses updated successfully",
	})
}

func (c *LabOrderController) wrapReadError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrLabOrderNotFound):
		return wrapError.Wrap(err, ctx, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrLabOrderFetchFailed, ctx, fiber.StatusInternalServerError)
	}
}

func (c *LabOrderController) wrapWriteError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, ctx, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrLabOrderNotFound):
		return wrapError.Wrap(err, ctx, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrLabOrderCreateFailed):
		return wrapError.Wrap(err, ctx, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrLabOrderUpdateFailed, ctx, fiber.StatusInternalServerError)
	}
}
