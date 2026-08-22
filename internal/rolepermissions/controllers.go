package rolepermissions

import (
	"hospital-backend/internal/rolepermissions/dto"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type RolePermissionServicer interface {
	FindModulesByRoleID(roleID string) (dto.RoleAccess, error)
}

type RolePermissionController struct {
	Service RolePermissionServicer
}

func NewRolePermissionController(service RolePermissionServicer) *RolePermissionController {
	return &RolePermissionController{Service: service}
}

func (c *RolePermissionController) FindModulesByRoleID(ctx *fiber.Ctx) error {
	roleID := strings.TrimSpace(ctx.Params("roleID"))
	if roleID == "" {
		roleID = strings.TrimSpace(ctx.Query("role_id"))
	}
	if roleID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    http.StatusBadRequest,
			"message": "role_id is required",
		})
	}

	access, err := c.Service.FindModulesByRoleID(roleID)
	if err != nil {
		return err
	}
	return ctx.JSON(fiber.Map{
		"code":    http.StatusOK,
		"message": "success",
		"data":    access,
	})
}
