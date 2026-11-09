package mysql

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type CleanupStore struct {
	database *gorm.DB
}

func NewCleanupStore(database *gorm.DB) CleanupStore {
	return CleanupStore{database: database}
}

func (store CleanupStore) DeleteExpired(ctx context.Context, currentTime time.Time) ([]string, error) {
	var codes []string

	err := store.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		err := transaction.Model(&model.Room{}).
			Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
			Where("expires_at <= ?", currentTime).
			Pluck("code", &codes).Error
		if err != nil {
			return fmt.Errorf("find expired rooms: %w", err)
		}

		if len(codes) == 0 {
			return nil
		}

		err = transaction.Where("code IN ?", codes).Delete(&model.Room{}).Error
		if err != nil {
			return fmt.Errorf("delete expired rooms: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return codes, nil
}
