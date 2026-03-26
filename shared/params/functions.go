package params

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

func New(c *fiber.Ctx) (*Payload, error) {
	m := make(map[string]any)
	err := json.Unmarshal(c.Body(), &m)
	if err != nil {
		return nil, err
	}
	return &Payload{Param: m}, nil
}
