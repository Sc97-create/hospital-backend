package middleware

import (
	"hospital-backend/internal/jwt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

func HandleMiddleware(app *fiber.App) {
	app.Use(RequestLogger()) // must be first — request-scoped zap logger + X-Request-ID
	app.Use(helmet.New())
	app.Use(logger.New())
	app.Use(healthcheck.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:9069,http://localhost:5173",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE",
		AllowHeaders:     "Content-Type,Origin,Accept,Authorization,Idempotency-Key,X-Request-ID",
		AllowCredentials: true,
	}))

}

func Authenticate(c *fiber.Ctx, jwtSvc *jwt.JwtService) error {
	log := GetLogger(c)
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		log.Warn("auth rejected",
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
		log.Warn("auth rejected",
			zap.String("reason", "invalid_format"),
			zap.String("path", c.Path()),
			zap.String("method", c.Method()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token format",
		})
	}
	flag, err := jwtSvc.ValidateAccessToken(parts[1])
	if err != nil || !flag {
		log.Warn("auth rejected",
			zap.String("reason", "invalid_or_expired"),
			zap.String("path", c.Path()),
			zap.String("method", c.Method()),
			zap.Error(err),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	return c.Next()
}
