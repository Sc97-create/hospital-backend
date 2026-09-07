package middleware

import (
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/permissions"
	rpdto "hospital-backend/internal/rolepermissions/dto"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

func HandleMiddleware(app *fiber.App) {
	app.Use(RequestLogger())
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
	userID, err := jwtSvc.AccessTokenSubject(parts[1])
	if err != nil {
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
	c.Locals(UserIDKey, userID)
	return c.Next()
}

// LoadRoleAccess loads the caller's role permission matrix into locals after Authenticate.
func LoadRoleAccess(c *fiber.Ctx, loader RoleAccessLoader, roleLookup RoleIDFinder) error {
	log := GetLogger(c)
	userID := GetUserID(c)
	if userID == "" || loader == nil || roleLookup == nil {
		log.Warn("rbac load rejected", zap.String("reason", "missing_deps_or_user"))
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	roleID, err := roleLookup.FindRoleIDByUserID(userID)
	if err != nil {
		log.Warn("rbac load rejected",
			zap.String("reason", "role_id_lookup"),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	access, err := loader.FindModulesByRoleID(roleID)
	if err != nil {
		log.Warn("rbac load rejected",
			zap.String("reason", "role_permissions"),
			zap.String("user_id", userID),
			zap.String("role_id", roleID),
			zap.Error(err),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	c.Locals(RoleAccessKey, toRoleAccessLocal(access))
	return c.Next()
}

func toRoleAccessLocal(access rpdto.RoleAccess) RoleAccessLocal {
	byModule := make(map[string]rpdto.ModulePermissionFlags, len(access.Permissions))
	for _, item := range access.Permissions {
		byModule[item.ModuleName] = item.Permissions
	}
	return RoleAccessLocal{IsAdmin: access.IsAdmin, ByModule: byModule}
}

func AuthorizeRBAC(c *fiber.Ctx) error {
	log := GetLogger(c)
	routePath := resolveRequestRoutePath(c)
	if IsPublicRoute(routePath) || IsPublicRoute(c.Path()) {
		return c.Next()
	}
	access, ok := c.Locals(RoleAccessKey).(RoleAccessLocal)
	if !ok {
		log.Warn("rbac denied",
			zap.String("reason", "missing_role_access"),
			zap.String("path", routePath),
			zap.String("user_id", GetUserID(c)),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	if access.IsAdmin {
		return c.Next()
	}
	module, action, ok := ResolveRoutePermission(routePath)
	if !ok {
		log.Warn("rbac denied",
			zap.String("reason", "unmapped_route"),
			zap.String("path", routePath),
			zap.String("user_id", GetUserID(c)),
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
	if hasModuleAction(access, module, action) {
		return c.Next()
	}
	log.Warn("rbac denied",
		zap.String("reason", "insufficient_permission"),
		zap.String("path", routePath),
		zap.String("module", module),
		zap.String("action", action),
		zap.String("user_id", GetUserID(c)),
	)
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
}

func resolveRequestRoutePath(c *fiber.Ctx) string {
	return normalizeRoutePath(c.Path())
}

func hasModuleAction(access RoleAccessLocal, module, action string) bool {
	flags, ok := access.ByModule[module]
	if !ok {
		return false
	}
	switch action {
	case permissions.Create:
		return flags.Create
	case permissions.Update:
		return flags.Update
	case permissions.View:
		return flags.View
	case permissions.Delete:
		return flags.Delete
	default:
		return false
	}
}

// UseProtected attaches Authenticate → LoadRoleAccess → AuthorizeRBAC on a route group.
func UseProtected(group fiber.Router, jwtSvc *jwt.JwtService, loader RoleAccessLoader, roleLookup RoleIDFinder) {
	group.Use(func(c *fiber.Ctx) error {
		return Authenticate(c, jwtSvc)
	})
	group.Use(func(c *fiber.Ctx) error {
		return LoadRoleAccess(c, loader, roleLookup)
	})
	group.Use(AuthorizeRBAC)
}
