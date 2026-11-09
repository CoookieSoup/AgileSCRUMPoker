package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type Repository interface {
	DeleteExpired(ctx context.Context, currentTime time.Time) ([]string, error)
}

type Publisher interface {
	CloseRoom(code string, reason model.CloseReason)
}

type Clock interface {
	Now() time.Time
}

type Logger interface {
	Error(message string, args ...any)
}

type Service struct {
	repository Repository
	publisher  Publisher
	clock      Clock
	logger     Logger
}

func New(repository Repository, publisher Publisher, clock Clock, logger Logger) Service {
	return Service{
		repository: repository,
		publisher:  publisher,
		clock:      clock,
		logger:     logger,
	}
}

func (service Service) Sweep(ctx context.Context) error {
	codes, err := service.repository.DeleteExpired(ctx, service.clock.Now())
	if err != nil {
		return fmt.Errorf("delete expired rooms: %w", err)
	}

	for _, code := range codes {
		service.publisher.CloseRoom(code, model.CloseReasonExpired)
	}

	return nil
}

func (service Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := service.Sweep(ctx)
			if err != nil {
				service.logger.Error("room cleanup failed", "error", err)
			}
		}
	}
}
