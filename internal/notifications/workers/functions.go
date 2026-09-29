package workers

import (
	"context"
	"hospital-backend/internal/notifications"
	"hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/notifications/repository"
	"hospital-backend/pkg/logger"
	"time"

	"go.uber.org/zap"
)

type Worker struct {
	repo    repository.Repository
	factory *notifications.Factory
}

func NewWorker(repo repository.Repository, factory *notifications.Factory) *Worker {
	return &Worker{repo: repo, factory: factory}
}

func (w *Worker) Start(ctx context.Context) {
	ensureLog(nil).Info("notification worker started", zap.String("interval", "1m"))
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			ensureLog(nil).Info("notification worker stopped", zap.String("reason", "context_done"))
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	log := ensureLog(nil)
	notificationList, err := w.repo.GetPending(ctx, 100)
	if err != nil {
		return
	}
	for _, n := range notificationList {
		w.deliver(ctx, log, n)
	}
}

func (w *Worker) deliver(ctx context.Context, log *zap.Logger, n notifications.Notification) {
	sender, err := w.factory.Get(n.ProviderPayload.Channel)
	if err != nil {
		log.Error("notification send failed",
			zap.String("notification_id", n.ID),
			zap.String("notification_type", n.NotificationType),
			zap.String("reason", "sender"),
			zap.Error(err),
		)
		return
	}
	err = sender.Send(ctx, dto.Request{
		Recipient: n.ProviderPayload.RecipientEmail,
		Subject:   n.ProviderPayload.Subject,
		Content:   n.ProviderPayload.Content,
	})
	if err != nil {
		log.Error("notification send failed",
			zap.String("notification_id", n.ID),
			zap.String("notification_type", n.NotificationType),
			zap.String("reason", "send"),
			zap.Error(err),
		)
		nextRetry := time.Now().Add(15 * time.Minute)
		if markErr := w.repo.MarkFailed(ctx, n.ID, nextRetry, err); markErr != nil {
			log.Error("notification send failed",
				zap.String("notification_id", n.ID),
				zap.String("reason", "mark_failed"),
				zap.Error(markErr),
			)
		}
		return
	}
	if err = w.repo.MarkSent(ctx, n.ID); err != nil {
		log.Error("notification send failed",
			zap.String("notification_id", n.ID),
			zap.String("reason", "mark_sent"),
			zap.Error(err),
		)
		return
	}
	log.Info("notification sent",
		zap.String("notification_id", n.ID),
		zap.String("notification_type", n.NotificationType),
	)
}

func ensureLog(log *zap.Logger) *zap.Logger {
	if log != nil {
		return log
	}
	if logger.Log != nil {
		return logger.Log
	}
	return zap.NewNop()
}
