package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type RoomStore struct {
	database *gorm.DB
}

func NewRoomStore(database *gorm.DB) RoomStore {
	return RoomStore{database: database}
}

func (store RoomStore) Create(ctx context.Context, room model.Room) error {
	err := store.database.WithContext(ctx).Create(&room).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return model.ErrCodeTaken
	}

	if err != nil {
		return fmt.Errorf("insert room: %w", err)
	}

	return nil
}

func (store RoomStore) AddParticipant(
	ctx context.Context,
	participant model.Participant,
	currentTime time.Time,
) error {
	return store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		room, err := lockRoom(transaction, participant.RoomCode)
		if err != nil {
			return err
		}

		if room.IsExpired(currentTime) {
			return model.ErrRoomNotFound
		}

		var count int64

		err = transaction.Model(&model.Participant{}).Where("room_code = ?", room.Code).Count(&count).Error
		if err != nil {
			return fmt.Errorf("count participants: %w", err)
		}

		if count >= model.MaxParticipants {
			return model.ErrRoomFull
		}

		err = transaction.Create(&participant).Error
		if err != nil {
			return fmt.Errorf("insert participant: %w", err)
		}

		return nil
	})
}

func (store RoomStore) Load(ctx context.Context, code string, currentTime time.Time) (model.Room, error) {
	return loadRoom(ctx, store.database, code, currentTime)
}

func (store RoomStore) RemoveParticipant(
	ctx context.Context,
	code string,
	participantID string,
) (model.Departure, error) {
	var departure model.Departure

	err := store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		_, err := lockRoom(transaction, code)
		if err != nil {
			return err
		}

		var leaving model.Participant

		err = transaction.Where("id = ? AND room_code = ?", participantID, code).Take(&leaving).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrUnauthorized
		}

		if err != nil {
			return fmt.Errorf("find participant: %w", err)
		}

		err = transaction.Delete(&leaving).Error
		if err != nil {
			return fmt.Errorf("delete participant: %w", err)
		}

		departure, err = settleRoom(transaction, code, leaving.IsHost)

		return err
	})
	if err != nil {
		return model.Departure{}, err
	}

	return departure, nil
}

func (store RoomStore) Delete(ctx context.Context, code string) error {
	result := store.database.WithContext(ctx).Where("code = ?", code).Delete(&model.Room{})
	if result.Error != nil {
		return fmt.Errorf("delete room: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return model.ErrRoomNotFound
	}

	return nil
}

func settleRoom(transaction *gorm.DB, code string, hostLeft bool) (model.Departure, error) {
	var successor model.Participant

	err := transaction.Where("room_code = ?", code).Order("created_at, id").Take(&successor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = transaction.Where("code = ?", code).Delete(&model.Room{}).Error
		if err != nil {
			return model.Departure{}, fmt.Errorf("delete empty room: %w", err)
		}

		return model.Departure{RoomDeleted: true}, nil
	}

	if err != nil {
		return model.Departure{}, fmt.Errorf("find successor: %w", err)
	}

	if !hostLeft {
		return model.Departure{}, nil
	}

	err = transaction.Model(&successor).Update("is_host", true).Error
	if err != nil {
		return model.Departure{}, fmt.Errorf("promote host: %w", err)
	}

	return model.Departure{NewHostID: successor.ID}, nil
}
