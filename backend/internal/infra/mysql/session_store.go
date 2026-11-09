package mysql

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type SessionStore struct {
	database *gorm.DB
}

func NewSessionStore(database *gorm.DB) SessionStore {
	return SessionStore{database: database}
}

func (store SessionStore) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (model.Participant, model.Room, error) {
	var participant model.Participant

	err := store.database.WithContext(ctx).Where("token_hash = ?", tokenHash).Take(&participant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Participant{}, model.Room{}, model.ErrUnauthorized
	}

	if err != nil {
		return model.Participant{}, model.Room{}, fmt.Errorf("find participant: %w", err)
	}

	var room model.Room

	err = store.database.WithContext(ctx).Where("code = ?", participant.RoomCode).Take(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Participant{}, model.Room{}, model.ErrUnauthorized
	}

	if err != nil {
		return model.Participant{}, model.Room{}, fmt.Errorf("find room: %w", err)
	}

	return participant, room, nil
}
