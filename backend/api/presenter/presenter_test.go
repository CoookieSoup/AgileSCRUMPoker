package presenter_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/presenter"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

func TestRoomStateWithRevealedRound(t *testing.T) {
	state := presenter.RoomState(model.RoomView{
		Code:         "ABC234",
		ExpiresAt:    time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC),
		Participants: []model.ParticipantView{{ID: "host", DisplayName: "Ona", IsHost: true, HasVoted: true}},
		Round: &model.RoundView{
			Label:    "Login",
			JiraKey:  "ABC-1",
			Revealed: true,
			Votes:    []model.VoteView{{ParticipantID: "host", Card: model.CardEight}},
		},
		OwnVote: model.CardEight,
	})

	encoded, err := json.Marshal(state)

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"code": "ABC234",
		"expiresAt": "2026-01-01T13:00:00Z",
		"participants": [{"id": "host", "displayName": "Ona", "isHost": true, "hasVoted": true}],
		"round": {
			"label": "Login",
			"jiraKey": "ABC-1",
			"revealed": true,
			"votes": [{"participantId": "host", "card": "8"}]
		},
		"ownVote": "8"
	}`, string(encoded))
}

func TestRoomStateOmitsEmptyOptionalFields(t *testing.T) {
	state := presenter.RoomState(model.RoomView{
		Code:  "ABC234",
		Round: &model.RoundView{Votes: []model.VoteView{}},
	})

	encoded, err := json.Marshal(state)

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"code": "ABC234",
		"expiresAt": "0001-01-01T00:00:00Z",
		"participants": [],
		"round": {"label": "", "revealed": false, "votes": []}
	}`, string(encoded))
}
