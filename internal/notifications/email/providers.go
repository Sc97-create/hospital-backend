package email

import (
	"context"
	"hospital-backend/internal/notifications/dto"
)

type Provider interface {
	Send(ctx context.Context, req dto.Request) error
}
