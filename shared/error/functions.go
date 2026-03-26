package error

import "github.com/gofiber/fiber/v2"

func Wrap(err error, c *fiber.Ctx, statuscode int) error {
	response := make(map[string]any)
	response["error"] = err
	response["code"] = statuscode
	return c.Status(statuscode).JSON(response)
}
func WrapV2(err error, code int) error {
	return fiber.NewError(code, err.Error())
}
