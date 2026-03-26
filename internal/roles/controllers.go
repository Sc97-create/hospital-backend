package roles

import (
	"hospital-backend/internal/roles/dto"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func FindMany(c *fiber.Ctx, service *RoleServices) error {
	var payload dto.FindManyRequest
	if err := c.QueryParser(&payload); err != nil {
		return err
	}
	if payload.Page == 0 {
		payload.Page = 1
	}
	offset := payload.Limit * (payload.Page - 1)
	roles, err := service.FindMany(payload.Limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"code":    http.StatusOK,
		"message": "success",
		"data":    roles,
	})
}
