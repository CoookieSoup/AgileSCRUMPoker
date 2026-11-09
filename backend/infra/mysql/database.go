package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

func Open(dsn string) (*gorm.DB, error) {
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return database, nil
}

func Migrate(database *gorm.DB) error {
	err := database.AutoMigrate(&model.Room{}, &model.Participant{}, &model.Round{}, &model.Vote{})
	if err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}

type Health struct {
	database *gorm.DB
}

func NewHealth(database *gorm.DB) Health {
	return Health{database: database}
}

func (health Health) Ping(ctx context.Context) error {
	connection, err := health.database.DB()
	if err != nil {
		return fmt.Errorf("get connection pool: %w", err)
	}

	err = connection.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}

func lockRoom(transaction *gorm.DB, code string) (model.Room, error) {
	var room model.Room

	err := transaction.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("code = ?", code).
		Take(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Room{}, model.ErrRoomNotFound
	}

	if err != nil {
		return model.Room{}, fmt.Errorf("lock room: %w", err)
	}

	return room, nil
}

func lockRound(transaction *gorm.DB, code string) (model.Round, error) {
	var round model.Round

	err := transaction.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("room_code = ?", code).
		Take(&round).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Round{}, model.ErrRoundNotFound
	}

	if err != nil {
		return model.Round{}, fmt.Errorf("lock round: %w", err)
	}

	return round, nil
}

func loadRoom(ctx context.Context, database *gorm.DB, code string, currentTime time.Time) (model.Room, error) {
	var room model.Room

	err := database.WithContext(ctx).
		Preload("Participants", func(query *gorm.DB) *gorm.DB {
			return query.Order("created_at, id")
		}).
		Preload("Round.Votes").
		Where("code = ? AND expires_at > ?", code, currentTime).
		Take(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Room{}, model.ErrRoomNotFound
	}

	if err != nil {
		return model.Room{}, fmt.Errorf("load room: %w", err)
	}

	return room, nil
}
