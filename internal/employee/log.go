package employee

import "go.uber.org/zap"

func ensureLog(log *zap.Logger) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}

func logDBError(log *zap.Logger, op string, err error) {
	ensureLog(log).Error("employee repo error", zap.String("op", op), zap.Error(err))
}
