package payments

import (
	"go.uber.org/zap"
)

type IWebhookRepository interface {
	CreateWebhookEvent(log *zap.Logger, webhook WebhookEvents) error
}

func (r *DB) CreateWebhookEvent(log *zap.Logger, webhook WebhookEvents) error {
	log = ensureLog(log)
	err := r.db.Create(&webhook).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "CreateWebhookEvent"), zap.Error(err))
		return err
	}
	return nil
}
