package mysql

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type RoundStore struct {
	database *gorm.DB
}

func NewRoundStore(database *gorm.DB) RoundStore {
	return RoundStore{database: database}
}

func (store RoundStore) Replace(ctx context.Context, round model.Round) error {
	return store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		_, err := lockRoom(transaction, round.RoomCode)
		if err != nil {
			return err
		}

		err = transaction.Where("room_code = ?", round.RoomCode).Delete(&model.Round{}).Error
		if err != nil {
			return fmt.Errorf("delete previous round: %w", err)
		}

		err = transaction.Create(&round).Error
		if err != nil {
			return fmt.Errorf("insert round: %w", err)
		}

		return nil
	})
}

func (store RoundStore) UpsertVote(
	ctx context.Context,
	code string,
	participantID string,
	card model.Card,
) error {
	return store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		round, err := lockRound(transaction, code)
		if err != nil {
			return err
		}

		if round.Revealed {
			return model.ErrRoundRevealed
		}

		vote := model.Vote{RoundID: round.ID, ParticipantID: participantID, Card: card}

		err = transaction.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"card"})}).
			Create(&vote).Error
		if err != nil {
			return fmt.Errorf("upsert vote: %w", err)
		}

		return nil
	})
}

func (store RoundStore) Reveal(ctx context.Context, code string) error {
	return store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		round, err := lockRound(transaction, code)
		if err != nil {
			return err
		}

		if round.Revealed {
			return nil
		}

		err = transaction.Model(&round).Update("revealed", true).Error
		if err != nil {
			return fmt.Errorf("reveal round: %w", err)
		}

		return nil
	})
}

func (store RoundStore) Load(ctx context.Context, code string, currentTime time.Time) (model.Room, error) {
	return loadRoom(ctx, store.database, code, currentTime)
}
