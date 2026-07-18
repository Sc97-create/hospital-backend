package repository

import (
	"context"
	"hospital-backend/internal/notifications"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, n *notifications.Notification) error
	GetPending(ctx context.Context, limit int) ([]notifications.Notification, error)
	MarkProcessing(ctx context.Context, id string) error
	MarkSent(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, nextRetryAt time.Time, err error) error
	CreateAttempt(ctx context.Context, attempt *notifications.NotificationAttempts) error
}

func (d Db) Create(ctx context.Context, n *notifications.Notification) error {
	return d.DB.WithContext(ctx).Create(n).Error
}

func (d Db) GetPending(ctx context.Context, limit int) ([]notifications.Notification, error) {
	var notificationArr []notifications.Notification
	err := d.DB.WithContext(ctx).Where("status = ?", notifications.PendingStatus).Limit(limit).Find(&notificationArr).Error
	return notificationArr, err
}

func (d Db) MarkProcessing(ctx context.Context, id string) error {
	return d.DB.WithContext(ctx).
		Model(&notifications.Notification{}).
		Where("id = ?", id).
		Update("status", notifications.ProcessingStatus).Error
}

func (d Db) MarkSent(ctx context.Context, id string) error {
	return d.DB.WithContext(ctx).
		Model(&notifications.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  notifications.SentStatus,
			"sent_at": time.Now(),
		}).Error
}

func (d Db) MarkFailed(ctx context.Context, id string, nextRetryAt time.Time, err error) error {
	errMsg := err.Error()
	return d.DB.WithContext(ctx).
		Model(&notifications.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        notifications.FailedStatus,
			"last_error":    errMsg,
			"next_retry_at": nextRetryAt,
			"retry_count":   gorm.Expr("retry_count + ?", 1),
		}).Error
}

func (d Db) CreateAttempt(ctx context.Context, attempt *notifications.NotificationAttempts) error {
	return d.DB.WithContext(ctx).Create(attempt).Error
}
