package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// RequireRole allows the request only when the authenticated user's role name matches allowed.
// Super Admin (is_admin) does not bypass this gate.
func RequireRole(roleLookup RoleIDFinder, names RoleNameFinder, allowed string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := GetLogger(c)
		userID := GetUserID(c)
		if strings.TrimSpace(allowed) == "" || !roleGateReady(userID, roleLookup, names) {
			return denyRole(c, log, "missing_deps_or_user", userID, nil)
		}
		roleID, err := roleLookup.FindRoleIDByUserID(log, userID)
		if err != nil || strings.TrimSpace(roleID) == "" {
			return denyRole(c, log, "role_id_lookup", userID, err)
		}
		name, err := names.FindNameByID(log, roleID)
		if err != nil || strings.TrimSpace(name) != allowed {
			return denyRole(c, log, "role_not_allowed", userID, err)
		}
		return c.Next()
	}
}

func roleGateReady(userID string, roleLookup RoleIDFinder, names RoleNameFinder) bool {
	return userID != "" && roleLookup != nil && names != nil
}

func denyRole(c *fiber.Ctx, log *zap.Logger, reason, userID string, err error) error {
	fields := []zap.Field{
		zap.String("reason", reason),
		zap.String("user_id", userID),
		zap.String("path", c.Path()),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	if log != nil {
		log.Warn("role gate rejected", fields...)
	}
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
}
