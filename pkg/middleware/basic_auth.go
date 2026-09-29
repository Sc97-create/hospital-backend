package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// AuthenticateBasic validates Authorization: Basic <base64(id:secret)> against the
// globally configured internal basic-auth credentials. Used by internal APIs only.
func AuthenticateBasic(c *fiber.Ctx, id, secret string) error {
	log := GetLogger(c)
	if strings.TrimSpace(id) == "" || strings.TrimSpace(secret) == "" {
		log.Error("basic auth rejected",
			zap.String("reason", "credentials_not_configured"),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		log.Warn("basic auth rejected",
			zap.String("reason", "missing_header"),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		log.Warn("basic auth rejected",
			zap.String("reason", "invalid_format"),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization format",
		})
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		log.Warn("basic auth rejected",
			zap.String("reason", "invalid_base64"),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization credentials",
		})
	}

	credParts := strings.SplitN(string(decoded), ":", 2)
	if len(credParts) != 2 {
		log.Warn("basic auth rejected",
			zap.String("reason", "invalid_credential_shape"),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization credentials",
		})
	}

	if !secureEqual(credParts[0], id) || !secureEqual(credParts[1], secret) {
		log.Warn("basic auth rejected",
			zap.String("reason", "mismatch"),
			zap.String("path", c.Path()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.Next()
}

func secureEqual(a, b string) bool {
	sumA := sha256.Sum256([]byte(a))
	sumB := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(sumA[:], sumB[:]) == 1
}
