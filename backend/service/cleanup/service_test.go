package cleanup_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/service/cleanup"
)

var (
	errStore    = errors.New("store failed")
	currentTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

type fakeRepository struct {
	codes []string
	err   error
}

func (repository fakeRepository) DeleteExpired(_ context.Context, moment time.Time) ([]string, error) {
	if !moment.Equal(currentTime) {
		return nil, errors.New("unexpected time")
	}

	return repository.codes, repository.err
}

type fakePublisher struct {
	mutex  sync.Mutex
	closed map[string]model.CloseReason
}

func (publisher *fakePublisher) CloseRoom(code string, reason model.CloseReason) {
	publisher.mutex.Lock()
	defer publisher.mutex.Unlock()

	publisher.closed[code] = reason
}

func (publisher *fakePublisher) count() int {
	publisher.mutex.Lock()
	defer publisher.mutex.Unlock()

	return len(publisher.closed)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return currentTime
}

type fakeLogger struct {
	mutex    sync.Mutex
	messages []string
}

func (logger *fakeLogger) Error(message string, _ ...any) {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	logger.messages = append(logger.messages, message)
}

func (logger *fakeLogger) count() int {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	return len(logger.messages)
}

func TestSweepClosesEveryExpiredRoom(t *testing.T) {
	publisher := &fakePublisher{closed: map[string]model.CloseReason{}}
	repository := fakeRepository{codes: []string{"ABC234", "XYZ789"}}
	service := cleanup.New(repository, publisher, fixedClock{}, &fakeLogger{})

	err := service.Sweep(context.Background())

	require.NoError(t, err)
	assert.Equal(t, map[string]model.CloseReason{
		"ABC234": model.CloseReasonExpired,
		"XYZ789": model.CloseReasonExpired,
	}, publisher.closed)
}

func TestSweepReturnsStoreError(t *testing.T) {
	publisher := &fakePublisher{closed: map[string]model.CloseReason{}}
	service := cleanup.New(fakeRepository{err: errStore}, publisher, fixedClock{}, &fakeLogger{})

	err := service.Sweep(context.Background())

	require.ErrorIs(t, err, errStore)
	assert.Empty(t, publisher.closed)
}

func TestRunSweepsUntilCancelled(t *testing.T) {
	publisher := &fakePublisher{closed: map[string]model.CloseReason{}}
	repository := fakeRepository{codes: []string{"ABC234"}}
	service := cleanup.New(repository, publisher, fixedClock{}, &fakeLogger{})
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})

	go func() {
		service.Run(ctx, time.Millisecond)
		close(finished)
	}()

	assert.Eventually(t, func() bool { return publisher.count() == 1 }, time.Second, time.Millisecond)
	cancel()
	assert.Eventually(t, func() bool {
		select {
		case <-finished:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
}

func TestRunLogsFailures(t *testing.T) {
	logger := &fakeLogger{}
	publisher := &fakePublisher{closed: map[string]model.CloseReason{}}
	service := cleanup.New(fakeRepository{err: errStore}, publisher, fixedClock{}, logger)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go service.Run(ctx, time.Millisecond)

	assert.Eventually(t, func() bool { return logger.count() > 0 }, time.Second, time.Millisecond)
}
