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
	err := d.DB.WithContext(ctx).Create(n).Error
	if err != nil {
		logDBError("Create", notificationID(n), err)
		return err
	}
	return nil
}

func notificationID(n *notifications.Notification) string {
	if n == nil {
		return ""
	}
	return n.ID
}

func (d Db) GetPending(ctx context.Context, limit int) ([]notifications.Notification, error) {
	var notificationArr []notifications.Notification
	err := d.DB.WithContext(ctx).Where("status = ?", notifications.PendingStatus).Limit(limit).Find(&notificationArr).Error
	if err != nil {
		logDBError("GetPending", "", err)
	}
	return notificationArr, err
}

func (d Db) MarkProcessing(ctx context.Context, id string) error {
	err := d.DB.WithContext(ctx).
		Model(&notifications.Notification{}).
		Where("id = ?", id).
		Update("status", notifications.ProcessingStatus).Error
	if err != nil {
		logDBError("MarkProcessing", id, err)
	}
	return err
}

func (d Db) MarkSent(ctx context.Context, id string) error {
	err := d.DB.WithContext(ctx).
		Model(&notifications.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  notifications.SentStatus,
			"sent_at": time.Now(),
		}).Error
	if err != nil {
		logDBError("MarkSent", id, err)
	}
	return err
}

func (d Db) MarkFailed(ctx context.Context, id string, nextRetryAt time.Time, err error) error {
	errMsg := err.Error()
	dbErr := d.DB.WithContext(ctx).
		Model(&notifications.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        notifications.FailedStatus,
			"last_error":    errMsg,
			"next_retry_at": nextRetryAt,
			"retry_count":   gorm.Expr("retry_count + ?", 1),
		}).Error
	if dbErr != nil {
		logDBError("MarkFailed", id, dbErr)
	}
	return dbErr
}

func (d Db) CreateAttempt(ctx context.Context, attempt *notifications.NotificationAttempts) error {
	err := d.DB.WithContext(ctx).Create(attempt).Error
	if err != nil {
		id := ""
		if attempt != nil {
			id = attempt.NotificationID
		}
		logDBError("CreateAttempt", id, err)
	}
	return err
}
