package round

import (
	"context"
	"time"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type Repository interface {
	Replace(ctx context.Context, round model.Round) error
	UpsertVote(ctx context.Context, code string, participantID string, card model.Card) error
	Reveal(ctx context.Context, code string) error
	Load(ctx context.Context, code string, currentTime time.Time) (model.Room, error)
}

type Publisher interface {
	Publish(code string, event model.Event)
}

type IDGenerator interface {
	NewID() (string, error)
}

type Clock interface {
	Now() time.Time
}
