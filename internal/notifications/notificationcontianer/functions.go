package notificationcontainer

import (
	"context"
	"hospital-backend/config"
	"hospital-backend/internal/notifications/module"
	"hospital-backend/internal/notifications/service"

	"gorm.io/gorm"
)

type NotificationContainer struct {
	Service *service.Notificationservice
	module  *module.Module
}

func NewNotificationContainer(db *gorm.DB, cfg config.Config) *NotificationContainer {

	mod, err := module.New(db, cfg)
	if err != nil {
		return nil
	}

	return &NotificationContainer{
		Service: mod.Service,
		module:  mod,
	}
}

func (c *NotificationContainer) Start(ctx context.Context) {
	c.module.Start(ctx)
}
