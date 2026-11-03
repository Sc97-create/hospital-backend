package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterHealth mounts GET /api/v1/health. Callers hit it; the server does not poll itself.
func RegisterHealth(app *fiber.App, db *gorm.DB) {
	app.Get("/api/v1/health", Health(db))
}

// Health reports whether the process can reach the database.
func Health(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := pingDB(c.Context(), db); err != nil {
			GetLogger(c).Warn("health check failed", zap.String("reason", "db_ping"), zap.Error(err))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unavailable"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	}
}

func pingDB(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
