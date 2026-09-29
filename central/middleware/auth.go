package middleware

import (
	"hospital-backend/pkg/logger"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	loggerKey = "req_logger"
	userIDKey = "user_id"
)

// TokenVerifier reads the subject from a central access token.
type TokenVerifier interface {
	AccessTokenSubject(token string) (string, error)
}

// Authenticate requires Authorization: Bearer <token> and stores the subject as user_id.
func Authenticate(verifier TokenVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := GetLogger(c)
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Warn("central auth rejected",
				zap.String("reason", "missing_header"),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Warn("central auth rejected",
				zap.String("reason", "invalid_format"),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token format",
			})
		}
		if verifier == nil {
			log.Error("central auth rejected", zap.String("reason", "verifier_missing"))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		userID, err := verifier.AccessTokenSubject(parts[1])
		if err != nil {
			log.Warn("central auth rejected",
				zap.String("reason", "invalid_or_expired"),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.Error(err),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		c.Locals(userIDKey, userID)
		return c.Next()
	}
}

// GetLogger returns the request logger set by the app request logger.
func GetLogger(c *fiber.Ctx) *zap.Logger {
	if l, ok := c.Locals(loggerKey).(*zap.Logger); ok {
		return l
	}
	return logger.Log
}

// GetUserID returns the customer id stored by Authenticate.
func GetUserID(c *fiber.Ctx) string {
	userID, _ := c.Locals(userIDKey).(string)
	return userID
}
