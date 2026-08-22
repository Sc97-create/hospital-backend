package permissions

import (
	"hospital-backend/internal/modules"

	"github.com/gofiber/fiber/v2"
)

type PermissionServicer interface {
	FindMany() ([]modules.Modules, []Permission, error)
}

func FindMany(c *fiber.Ctx, service PermissionServicer) error {
	modules, permissions, err := service.FindMany()
	if err != nil {
		return err
	}
	response := make(map[string]any)
	response["modules"] = modules
	response["permissions"] = permissions
	response["code"] = 200
	return c.JSON(response)
}
