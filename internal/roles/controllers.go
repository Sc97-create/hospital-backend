package roles

import (
	"hospital-backend/internal/roles/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type RoleControllers interface {
	FindMany(c *fiber.Ctx) error
}

type RoleServicer interface {
	FindMany(log *zap.Logger, organisationID string, limit int, offset int) ([]dto.RoleResponse, int64, error)
}

type RoleController struct {
	RoleService RoleServicer
}

func NewRoleControllerInterface(roleService RoleServicer) *RoleController {
	return &RoleController{RoleService: roleService}
}

func (r *RoleController) FindMany(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	var payload dto.FindManyRequest
	if err := c.QueryParser(&payload); err != nil {
		logger.Warn("role list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if payload.Page == 0 {
		payload.Page = 1
	}
	logger.Info("role list attempt",
		zap.String("organisation_id", payload.OrganisationID),
		zap.Int("limit", payload.Limit),
		zap.Int("page", payload.Page),
	)
	offset := payload.Limit * (payload.Page - 1)
	roles, total, err := r.RoleService.FindMany(logger, payload.OrganisationID, payload.Limit, offset)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrRolesFetchFailed, c, fiber.StatusInternalServerError)
	}
	return c.JSON(fiber.Map{
		"code":    http.StatusOK,
		"message": "success",
		"data":    roles,
		"total":   total,
	})
}
