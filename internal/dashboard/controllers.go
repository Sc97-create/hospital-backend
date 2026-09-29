package dashboard

import (
	"hospital-backend/internal/dashboard/dto"
	"hospital-backend/pkg/middleware"
	errWrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	statusOk      = "200"
	dashboardData = "dashboard data retrieved successfully"
)

type DashboardController struct {
	service DashboardServicer
}

func NewDashboardController(service DashboardServicer) *DashboardController {
	return &DashboardController{service: service}
}

func (d *DashboardController) GetAppointmentsGroupedByStatus(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		logger.Warn("dashboard status counts request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("dashboard status counts attempt", zap.String("organisation_id", organisationID))

	grouped, err := d.service.GetAppointmentsGroupedByStatus(logger, organisationID)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(dto.Response{
		Data:    grouped,
		Message: dashboardData,
		Code:    statusOk,
	})
}

func (d *DashboardController) GetTodayLatestAppointments(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		logger.Warn("dashboard today appointments request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("dashboard today appointments attempt", zap.String("organisation_id", organisationID))

	appointments, err := d.service.GetTodayLatestAppointments(logger, organisationID)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(dto.Response{
		Data:    appointments,
		Message: dashboardData,
		Code:    statusOk,
	})
}

func (d *DashboardController) GetTodayCompletedInvoiceSummary(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		logger.Warn("dashboard invoice summary request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("dashboard invoice summary attempt", zap.String("organisation_id", organisationID))

	summary, err := d.service.GetTodayCompletedInvoiceSummary(logger, organisationID)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(dto.Response{
		Data:    summary,
		Message: dashboardData,
		Code:    statusOk,
	})
}

func (d *DashboardController) GetEmployeeStatusCounts(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		logger.Warn("dashboard employee status counts request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("dashboard employee status counts attempt", zap.String("organisation_id", organisationID))

	counts, err := d.service.GetEmployeeStatusCounts(logger, organisationID)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(dto.Response{
		Data:    counts,
		Message: dashboardData,
		Code:    statusOk,
	})
}

func (d *DashboardController) GetTodayPrescriptions(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		logger.Warn("dashboard today prescriptions request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("dashboard today prescriptions attempt", zap.String("organisation_id", organisationID))

	summary, err := d.service.GetTodayPrescriptions(logger, organisationID)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(dto.Response{
		Data:    summary,
		Message: dashboardData,
		Code:    statusOk,
	})
}
