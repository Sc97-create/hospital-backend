package department

import (
	"hospital-backend/internal/department/dto"

	"github.com/gofiber/fiber/v2"
)

func FindMany(c *fiber.Ctx, service *DepartmentService) error {
	payload := dto.FindManyRequest{}
	if err := c.QueryParser(&payload); err != nil {
		return err
	}
	if payload.Page == 0 {
		payload.Page = 1
	}
	offset := payload.Limit * (payload.Page - 1)
	department, err := service.FindMany(payload.Limit, offset)
	if err != nil {
		return err
	}
	response := make(map[string]interface{})
	response["data"] = department
	response["code"] = 200
	return c.JSON(response)
}
