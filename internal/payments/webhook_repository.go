package payments

type IWebhookRepository interface {
	CreateWebhookEvent(webhook WebhookEvents) error
}

func (r *DB) CreateWebhookEvent(webhook WebhookEvents) error {
	err := r.db.Create(&webhook).Error
	if err != nil {
		return err
	}
	return nil
}
