package payments

type IWebhookRepository interface {
	CreateWebhookEvent(webhook WebhookEvents) error
}

func (r *DB) CreateWebhookEvent(webhook WebhookEvents) error {
	return r.db.Create(webhook).Error
}
