package module

import (
	"context"
	"fmt"
	"hospital-backend/config"
	"hospital-backend/internal/notifications"
	"hospital-backend/internal/notifications/email"
	"hospital-backend/internal/notifications/render"
	"hospital-backend/internal/notifications/repository"
	"hospital-backend/internal/notifications/service"
	"hospital-backend/internal/notifications/workers"

	"gorm.io/gorm"
)

type Module struct {
	Service *service.Notificationservice

	worker *workers.Worker
}

func (m *Module) Start(ctx context.Context) {
	go m.worker.Start(ctx)
}

func New(db *gorm.DB, cfg config.Config) (*Module, error) {
	subjects := map[string]string{
		"Patient Created":     "Patient Created",
		"Appointment Created": "Appointment Created",
	}
	renderer, err := render.NewHTMLRenderer(cfg.TemplatePath, subjects)
	if err != nil {
		return nil, err
	}

	repo := repository.NewDB(db)

	smtpProvider, err := email.NewSmtpProvider(cfg.NotificationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP provider: %w", err)
	}

	emailSender := email.NewSender(smtpProvider)

	factory := notifications.NewFactory(map[notifications.ChannelType]notifications.Sender{
		notifications.ChannelType(notifications.EmailChannel): emailSender,
	})

	svc := service.NewNotificationService(repo, renderer)
	worker := workers.NewWorker(repo, factory)
	return &Module{
		Service: svc,
		worker:  worker,
	}, nil
}
