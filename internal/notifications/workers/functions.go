package workers

import (
	"context"
	"hospital-backend/internal/notifications"
	"hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/notifications/repository"
	"time"
)

type Worker struct {
	repo    repository.Repository
	factory *notifications.Factory
}

func NewWorker(repo repository.Repository, factory *notifications.Factory) *Worker {
	return &Worker{repo: repo, factory: factory}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	notificationList, err := w.repo.GetPending(ctx, 100)
	if err != nil {
		return
	}
	for _, n := range notificationList {
		sender, err := w.factory.Get(n.Channel)
		if err != nil {
			continue
		}
		err = sender.Send(ctx, dto.Request{
			Recipient: n.ID,
			Subject:   n.NotificationType,
			Content:   n.Content,
		})
		if err != nil {
			nextRetry := time.Now().Add(15 * time.Minute)
			_ = w.repo.MarkFailed(ctx, n.ID, nextRetry, err)
			continue
		}
		_ = w.repo.MarkSent(ctx, n.ID)
	}
}
