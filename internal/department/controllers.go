package department

import (
	"hospital-backend/internal/department/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type DepartmentControllers interface {
	FindMany(c *fiber.Ctx) error
}
type DepartmentServicer interface {
	FindMany(log *zap.Logger, organisationID string, limit int, skip int) ([]Department, int64, error)
}

type DepartmentController struct {
	DepartmentService DepartmentServicer
}

func NewDepartmentControllerInterface(departmentService DepartmentServicer) *DepartmentController {
	return &DepartmentController{DepartmentService: departmentService}
}

func (d *DepartmentController) FindMany(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload := dto.FindManyRequest{}
	if err := c.QueryParser(&payload); err != nil {
		logger.Warn("department list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if payload.Page == 0 {
		payload.Page = 1
	}
	logger.Info("department list attempt",
		zap.String("organisation_id", payload.OrganisationID),
		zap.Int("limit", payload.Limit),
		zap.Int("page", payload.Page),
	)
	offset := payload.Limit * (payload.Page - 1)
	department, total, err := d.DepartmentService.FindMany(logger, payload.OrganisationID, payload.Limit, offset)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrDepartmentsFetchFailed, c, fiber.StatusInternalServerError)
	}
	response := make(map[string]interface{})
	response["data"] = department
	response["total"] = total
	response["code"] = 200
	return c.JSON(response)
}
