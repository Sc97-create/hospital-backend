package permissions

import (
	"hospital-backend/internal/modules"
	"hospital-backend/pkg/logger"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// reqLoggerKey matches middleware.LoggerKey. This package cannot import
// middleware, because middleware already imports permissions.
const reqLoggerKey = "req_logger"

type PermissionControllers interface {
	FindMany(c *fiber.Ctx) error
}

type PermissionServicer interface {
	FindMany(log *zap.Logger) ([]modules.Modules, []Permission, error)
}

type PermissionController struct {
	Service PermissionServicer
}

func NewPermissionController(service PermissionServicer) *PermissionController {
	return &PermissionController{Service: service}
}

func (p *PermissionController) FindMany(c *fiber.Ctx) error {
	log := requestLogger(c)
	log.Info("permission list attempt")
	modules, permissions, err := p.Service.FindMany(log)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrPermissionsFetchFailed, c, fiber.StatusInternalServerError)
	}
	response := make(map[string]any)
	response["modules"] = modules
	response["permissions"] = permissions
	response["code"] = 200
	return c.JSON(response)
}

func requestLogger(c *fiber.Ctx) *zap.Logger {
	if log, ok := c.Locals(reqLoggerKey).(*zap.Logger); ok && log != nil {
		return log
	}
	if logger.Log != nil {
		return logger.Log
	}
	return ensureLog(nil)
}
