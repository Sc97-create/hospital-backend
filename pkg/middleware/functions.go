package middleware

import (
	"hospital-backend/internal/jwt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func HandleMiddleware(app *fiber.App) {
	app.Use(helmet.New())
	app.Use(logger.New())
	app.Use(healthcheck.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:9069,http://localhost:5173",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE",
		AllowHeaders:     "Content-Type,Origin,Accept,Authorization",
		AllowCredentials: true,
	}))

}
func Authenticate(c *fiber.Ctx, jwt *jwt.JwtService) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token format",
		})
	}
	flag, err := jwt.ValidateAccessToken(parts[1])
	if err != nil || !flag {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}
	//c.Locals("userID", jwtSub)

	return c.Next()
}
