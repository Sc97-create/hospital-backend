package repository

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

func logDBError(op, id string, err error) {
	ensureLog(nil).Error("notification repo error",
		zap.String("op", op),
		zap.String("notification_id", id),
		zap.Error(err),
	)
}
