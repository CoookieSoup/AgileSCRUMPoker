package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/service/session"
)

var currentTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

type fakeRepository struct {
	participants map[string]model.Participant
	expiresAt    time.Time
}

func (repository fakeRepository) FindByTokenHash(
	_ context.Context,
	tokenHash string,
) (model.Participant, model.Room, error) {
	participant, found := repository.participants[tokenHash]
	if !found {
		return model.Participant{}, model.Room{}, model.ErrUnauthorized
	}

	return participant, model.Room{Code: participant.RoomCode, ExpiresAt: repository.expiresAt}, nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return currentTime
}

func newService(expiresAt time.Time) session.Service {
	repository := fakeRepository{
		participants: map[string]model.Participant{
			model.HashToken("good-token"): {ID: "guest", RoomCode: "ABC234"},
		},
		expiresAt: expiresAt,
	}

	return session.New(repository, fixedClock{})
}

func TestAuthenticate(t *testing.T) {
	testCases := []struct {
		name      string
		code      string
		token     string
		expiresAt time.Time
		expected  error
	}{
		{name: "valid session", code: "abc234", token: "good-token", expiresAt: currentTime.Add(time.Minute)},
		{name: "missing token", code: "ABC234", token: "", expected: model.ErrUnauthorized},
		{name: "unknown token", code: "ABC234", token: "bad-token", expected: model.ErrUnauthorized},
		{
			name:      "token for another room",
			code:      "XYZ789",
			token:     "good-token",
			expiresAt: currentTime.Add(time.Minute),
			expected:  model.ErrUnauthorized,
		},
		{name: "expired room", code: "ABC234", token: "good-token", expiresAt: currentTime, expected: model.ErrRoomNotFound},
		{name: "malformed code", code: "??", token: "good-token", expected: model.ErrRoomNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			participant, err := newService(testCase.expiresAt).Authenticate(
				context.Background(),
				testCase.code,
				testCase.token,
			)
			if testCase.expected != nil {
				require.ErrorIs(t, err, testCase.expected)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, "guest", participant.ID)
		})
	}
}
