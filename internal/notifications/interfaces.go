package notifications

import (
	"context"
	"hospital-backend/internal/notifications/dto"
)

//contract

type Sender interface {
	Send(ctx context.Context, req dto.Request) error
}
