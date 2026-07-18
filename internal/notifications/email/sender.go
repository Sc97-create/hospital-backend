package email

import (
	"context"
	"hospital-backend/internal/notifications/dto"
)

type Sender struct {
	provider Provider
}

func NewSender(provider Provider) *Sender {
	return &Sender{
		provider: provider,
	}
}
func (s *Sender) Send(ctx context.Context, req dto.Request) error {
	return s.provider.Send(
		ctx,
		req,
	)
}
