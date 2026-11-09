package round_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/service/round"
)

var errStore = errors.New("store failed")

type storedVote struct {
	code          string
	participantID string
	card          model.Card
}

type fakeRepository struct {
	replaced  []model.Round
	replace   error
	votes     []storedVote
	voteErr   error
	revealed  []string
	revealErr error
	loaded    model.Room
}

func (repository *fakeRepository) Replace(_ context.Context, replaced model.Round) error {
	repository.replaced = append(repository.replaced, replaced)

	return repository.replace
}

func (repository *fakeRepository) UpsertVote(
	_ context.Context,
	code string,
	participantID string,
	card model.Card,
) error {
	repository.votes = append(repository.votes, storedVote{code: code, participantID: participantID, card: card})

	return repository.voteErr
}

func (repository *fakeRepository) Reveal(_ context.Context, code string) error {
	repository.revealed = append(repository.revealed, code)

	return repository.revealErr
}

func (repository *fakeRepository) Load(_ context.Context, _ string, _ time.Time) (model.Room, error) {
	return repository.loaded, nil
}

type fakePublisher struct {
	events []model.Event
}

func (publisher *fakePublisher) Publish(_ string, event model.Event) {
	publisher.events = append(publisher.events, event)
}

type fakeGenerator struct{}

func (fakeGenerator) NewID() (string, error) {
	return "round-1", nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
}

var (
	host  = model.Participant{ID: "host", RoomCode: "ABC234", IsHost: true}
	guest = model.Participant{ID: "guest", RoomCode: "ABC234"}
)

func newService() (round.Service, *fakeRepository, *fakePublisher) {
	repository := &fakeRepository{}
	publisher := &fakePublisher{}

	return round.New(repository, publisher, fakeGenerator{}, fixedClock{}), repository, publisher
}

func TestStartRequiresHost(t *testing.T) {
	service, repository, publisher := newService()

	_, err := service.Start(context.Background(), guest, "Login", "")

	require.ErrorIs(t, err, model.ErrNotHost)
	assert.Empty(t, repository.replaced)
	assert.Empty(t, publisher.events)
}

func TestStartValidatesInput(t *testing.T) {
	service, repository, _ := newService()

	_, err := service.Start(context.Background(), host, "Login", "not a key")

	require.ErrorIs(t, err, model.ErrInvalidJiraKey)
	assert.Empty(t, repository.replaced)
}

func TestStartReplacesRound(t *testing.T) {
	service, repository, publisher := newService()
	repository.loaded = model.Room{Code: "ABC234", Round: &model.Round{Label: "Login", JiraKey: "ABC-1"}}

	view, err := service.Start(context.Background(), host, " Login ", "abc-1")

	require.NoError(t, err)
	assert.Equal(t, []model.Round{{ID: "round-1", RoomCode: "ABC234", Label: "Login", JiraKey: "ABC-1"}},
		repository.replaced)
	require.NotNil(t, view.Round)
	assert.Equal(t, "Login", view.Round.Label)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, model.EventRoundStarted, publisher.events[0].Type)
}

func TestVoteNeverPublishesCard(t *testing.T) {
	service, repository, publisher := newService()

	err := service.Vote(context.Background(), guest, "13")

	require.NoError(t, err)
	assert.Equal(t, []storedVote{{code: "ABC234", participantID: "guest", card: model.CardThirteen}}, repository.votes)
	require.Len(t, publisher.events, 1)

	encoded, err := json.Marshal(publisher.events[0])

	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"participant_voted","payload":{"participantId":"guest"}}`, string(encoded))
}

func TestVoteFailures(t *testing.T) {
	testCases := []struct {
		name     string
		card     string
		voteErr  error
		expected error
	}{
		{name: "card outside deck", card: "4", expected: model.ErrInvalidCard},
		{name: "no round", card: "5", voteErr: model.ErrRoundNotFound, expected: model.ErrRoundNotFound},
		{name: "already revealed", card: "5", voteErr: model.ErrRoundRevealed, expected: model.ErrRoundRevealed},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service, repository, publisher := newService()
			repository.voteErr = testCase.voteErr

			err := service.Vote(context.Background(), guest, testCase.card)

			require.ErrorIs(t, err, testCase.expected)
			assert.Empty(t, publisher.events)
		})
	}
}

func TestRevealRequiresHost(t *testing.T) {
	service, repository, _ := newService()

	_, err := service.Reveal(context.Background(), guest)

	require.ErrorIs(t, err, model.ErrNotHost)
	assert.Empty(t, repository.revealed)
}

func TestRevealPublishesVotes(t *testing.T) {
	service, repository, publisher := newService()
	repository.loaded = model.Room{
		Code:         "ABC234",
		Participants: []model.Participant{host, guest},
		Round: &model.Round{
			Revealed: true,
			Votes:    []model.Vote{{ParticipantID: "guest", Card: model.CardEight}},
		},
	}

	view, err := service.Reveal(context.Background(), host)

	require.NoError(t, err)
	assert.Equal(t, []string{"ABC234"}, repository.revealed)
	assert.Equal(t, []model.VoteView{{ParticipantID: "guest", Card: model.CardEight}}, view.Round.Votes)
	require.Len(t, publisher.events, 1)
	assert.Equal(t, model.VotesRevealedEvent(view.Round.Votes), publisher.events[0])
}

func TestRevealStoreFailurePublishesNothing(t *testing.T) {
	service, repository, publisher := newService()
	repository.revealErr = errStore

	_, err := service.Reveal(context.Background(), host)

	require.ErrorIs(t, err, errStore)
	assert.Empty(t, publisher.events)
}
