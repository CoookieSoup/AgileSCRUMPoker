package session

import (
	"context"
	"fmt"
	"time"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type Repository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (model.Participant, model.Room, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	repository Repository
	clock      Clock
}

func New(repository Repository, clock Clock) Service {
	return Service{repository: repository, clock: clock}
}

func (service Service) Authenticate(ctx context.Context, code string, token string) (model.Participant, error) {
	roomCode, err := model.NormalizeCode(code)
	if err != nil {
		return model.Participant{}, err
	}

	if token == "" {
		return model.Participant{}, model.ErrUnauthorized
	}

	participant, room, err := service.repository.FindByTokenHash(ctx, model.HashToken(token))
	if err != nil {
		return model.Participant{}, fmt.Errorf("find session: %w", err)
	}

	if participant.RoomCode != roomCode {
		return model.Participant{}, model.ErrUnauthorized
	}

	if room.IsExpired(service.clock.Now()) {
		return model.Participant{}, model.ErrRoomNotFound
	}

	return participant, nil
}
