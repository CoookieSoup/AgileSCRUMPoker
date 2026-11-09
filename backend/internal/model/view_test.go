package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

func sampleRoom(revealed bool) model.Room {
	return model.Room{
		Code:      "ABC234",
		ExpiresAt: time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC),
		Participants: []model.Participant{
			{ID: "host", DisplayName: "Ona", IsHost: true},
			{ID: "guest", DisplayName: "Jonas"},
			{ID: "quiet", DisplayName: "Rūta"},
		},
		Round: &model.Round{
			ID:       "round",
			Label:    "Login page",
			JiraKey:  "ABC-1",
			Revealed: revealed,
			Votes: []model.Vote{
				{RoundID: "round", ParticipantID: "guest", Card: model.CardEight},
				{RoundID: "round", ParticipantID: "host", Card: model.CardThree},
			},
		},
	}
}

func TestNewRoomViewHidesVotesBeforeReveal(t *testing.T) {
	view := model.NewRoomView(sampleRoom(false), "guest")

	require.NotNil(t, view.Round)
	assert.Empty(t, view.Round.Votes)
	assert.False(t, view.Round.Revealed)
	assert.Equal(t, model.CardEight, view.OwnVote)
	assert.Equal(t, []bool{true, true, false}, votedFlags(view))
}

func TestNewRoomViewShowsVotesAfterReveal(t *testing.T) {
	view := model.NewRoomView(sampleRoom(true), "quiet")

	require.NotNil(t, view.Round)
	assert.Equal(t, []model.VoteView{
		{ParticipantID: "host", Card: model.CardThree},
		{ParticipantID: "guest", Card: model.CardEight},
	}, view.Round.Votes)
	assert.Empty(t, view.OwnVote)
}

func TestNewRoomViewWithoutRound(t *testing.T) {
	room := sampleRoom(false)
	room.Round = nil

	view := model.NewRoomView(room, "host")

	assert.Nil(t, view.Round)
	assert.Empty(t, view.OwnVote)
	assert.Equal(t, []bool{false, false, false}, votedFlags(view))
	assert.Equal(t, "ABC234", view.Code)
}

func TestRoomIsExpired(t *testing.T) {
	room := sampleRoom(false)

	assert.False(t, room.IsExpired(room.ExpiresAt.Add(-time.Second)))
	assert.True(t, room.IsExpired(room.ExpiresAt))
}

func TestRoomParticipant(t *testing.T) {
	room := sampleRoom(false)

	participant, found := room.Participant("guest")

	assert.True(t, found)
	assert.Equal(t, "Jonas", participant.DisplayName)

	_, found = room.Participant("missing")

	assert.False(t, found)
}

func votedFlags(view model.RoomView) []bool {
	flags := make([]bool, 0, len(view.Participants))
	for _, participant := range view.Participants {
		flags = append(flags, participant.HasVoted)
	}

	return flags
}
