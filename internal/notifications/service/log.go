package service

import (
	"hospital-backend/pkg/logger"

	"go.uber.org/zap"
)

func ensureLog(log *zap.Logger) *zap.Logger {
	if log != nil {
		return log
	}
	if logger.Log != nil {
		return logger.Log
	}
	return zap.NewNop()
}
