package middleware

import (
	"hospital-backend/pkg/logger"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const LoggerKey = "req_logger"
const RequestIDKey = "request_id"

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := uuid.NewString()
		start := time.Now()

		reqLogger := logger.Log.With(
			zap.String("request_id", requestID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
		)

		c.Locals(LoggerKey, reqLogger)
		c.Locals(RequestIDKey, requestID)

		c.Set("X-Request-ID", requestID)

		reqLogger.Info("request started")

		err := c.Next()

		latency := time.Since(start)
		statusCode := c.Response().StatusCode()

		if err != nil || statusCode >= 500 {
			reqLogger.Error("request failed",
				zap.Int("status", statusCode),
				zap.Duration("latency_ms", latency),
				zap.Error(err),
			)
		} else if statusCode >= 400 {
			reqLogger.Warn("client error",
				zap.Int("status", statusCode),
				zap.Duration("latency_ms", latency),
			)
		} else {
			reqLogger.Info("request completed",
				zap.Int("status", statusCode),
				zap.Duration("latency_ms", latency),
			)
		}

		return err
	}
}

// GetLogger extracts the request-scoped logger from Fiber context.
// Falls back to the global logger when called outside an HTTP request (e.g. background jobs).
func GetLogger(c *fiber.Ctx) *zap.Logger {
	if l, ok := c.Locals(LoggerKey).(*zap.Logger); ok {
		return l
	}
	return logger.Log
}
