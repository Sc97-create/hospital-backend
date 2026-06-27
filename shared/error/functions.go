package error

import "github.com/gofiber/fiber/v2"

func Wrap(err error, c *fiber.Ctx, statuscode int) error {
	return c.Status(statuscode).JSON(fiber.Map{
		"error": err.Error(),
		"code":  statuscode,
	})
}
