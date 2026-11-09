package room

import (
	"context"
	"time"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type Repository interface {
	Create(ctx context.Context, room model.Room) error
	AddParticipant(ctx context.Context, participant model.Participant, currentTime time.Time) error
	Load(ctx context.Context, code string, currentTime time.Time) (model.Room, error)
	RemoveParticipant(ctx context.Context, code string, participantID string) (model.Departure, error)
	Delete(ctx context.Context, code string) error
}

type Publisher interface {
	Publish(code string, event model.Event)
	CloseRoom(code string, reason model.CloseReason)
}

type Generator interface {
	NewCode() (string, error)
	NewID() (string, error)
	NewToken() (string, error)
}

type Clock interface {
	Now() time.Time
}
