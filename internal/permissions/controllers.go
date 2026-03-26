package permissions

import "github.com/gofiber/fiber/v2"

func FindMany(c *fiber.Ctx, service *PermService) error {
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
