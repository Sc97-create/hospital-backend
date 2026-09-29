package rolepermissions

import (
	"hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type RolePermissionServicer interface {
	FindModulesByRoleID(log *zap.Logger, roleID string) (dto.RoleAccess, error)
}

type RolePermissionController struct {
	Service RolePermissionServicer
}

func NewRolePermissionController(service RolePermissionServicer) *RolePermissionController {
	return &RolePermissionController{Service: service}
}

func (c *RolePermissionController) FindModulesByRoleID(ctx *fiber.Ctx) error {
	logger := middleware.GetLogger(ctx)
	roleID := strings.TrimSpace(ctx.Params("roleID"))
	if roleID == "" {
		roleID = strings.TrimSpace(ctx.Query("role_id"))
	}
	if roleID == "" {
		logger.Warn("role permission request invalid", zap.String("reason", "missing_role_id"))
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    http.StatusBadRequest,
			"message": "role_id is required",
		})
	}
	logger.Info("role permission lookup attempt", zap.String("role_id", roleID))

	access, err := c.Service.FindModulesByRoleID(logger, roleID)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrRolePermissionsFetchFailed, ctx, fiber.StatusInternalServerError)
	}
	return ctx.JSON(fiber.Map{
		"code":    http.StatusOK,
		"message": "success",
		"data":    access,
	})
}
