package room_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/service/room"
)

var errStore = errors.New("store failed")

type fakeRepository struct {
	createErrors []error
	created      []model.Room
	added        []model.Participant
	addErr       error
	loaded       model.Room
	loadErr      error
	departure    model.Departure
	removeErr    error
	deleted      []string
	deleteErr    error
}

func (repository *fakeRepository) Create(_ context.Context, created model.Room) error {
	repository.created = append(repository.created, created)
	if len(repository.createErrors) == 0 {
		return nil
	}

	err := repository.createErrors[0]
	repository.createErrors = repository.createErrors[1:]

	return err
}

func (repository *fakeRepository) AddParticipant(_ context.Context, participant model.Participant, _ time.Time) error {
	repository.added = append(repository.added, participant)

	return repository.addErr
}

func (repository *fakeRepository) Load(_ context.Context, _ string, _ time.Time) (model.Room, error) {
	return repository.loaded, repository.loadErr
}

func (repository *fakeRepository) RemoveParticipant(_ context.Context, _ string, _ string) (model.Departure, error) {
	return repository.departure, repository.removeErr
}

func (repository *fakeRepository) Delete(_ context.Context, code string) error {
	repository.deleted = append(repository.deleted, code)

	return repository.deleteErr
}

type fakePublisher struct {
	events []model.Event
	closed []model.CloseReason
}

func (publisher *fakePublisher) Publish(_ string, event model.Event) {
	publisher.events = append(publisher.events, event)
}

func (publisher *fakePublisher) CloseRoom(_ string, reason model.CloseReason) {
	publisher.closed = append(publisher.closed, reason)
}

type fakeGenerator struct {
	codes   []string
	counter int
}

func (generator *fakeGenerator) NewCode() (string, error) {
	code := generator.codes[0]
	if len(generator.codes) > 1 {
		generator.codes = generator.codes[1:]
	}

	return code, nil
}

func (generator *fakeGenerator) NewID() (string, error) {
	generator.counter++

	return fmt.Sprintf("id-%d", generator.counter), nil
}

func (generator *fakeGenerator) NewToken() (string, error) {
	generator.counter++

	return fmt.Sprintf("token-%d", generator.counter), nil
}

type fixedClock struct {
	moment time.Time
}

func (clock fixedClock) Now() time.Time {
	return clock.moment
}

type fixture struct {
	repository *fakeRepository
	publisher  *fakePublisher
	service    room.Service
}

var startTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func newFixture(codes ...string) fixture {
	if len(codes) == 0 {
		codes = []string{"ABC234"}
	}

	repository := &fakeRepository{}
	publisher := &fakePublisher{}
	generator := &fakeGenerator{codes: codes}

	return fixture{
		repository: repository,
		publisher:  publisher,
		service:    room.New(repository, publisher, generator, fixedClock{moment: startTime}),
	}
}

func TestCreateMakesCallerHost(t *testing.T) {
	setup := newFixture()

	session, err := setup.service.Create(context.Background(), "  Ona ")

	require.NoError(t, err)
	require.Len(t, setup.repository.created, 1)

	created := setup.repository.created[0]

	assert.Equal(t, "ABC234", created.Code)
	assert.Equal(t, startTime.Add(time.Hour), created.ExpiresAt)
	require.Len(t, created.Participants, 1)
	assert.True(t, created.Participants[0].IsHost)
	assert.Equal(t, "Ona", created.Participants[0].DisplayName)
	assert.Equal(t, "ABC234", created.Participants[0].RoomCode)
	assert.Equal(t, model.HashToken(session.Token), created.Participants[0].TokenHash)
	assert.Equal(t, created.Participants[0].ID, session.ParticipantID)
	assert.Equal(t, "ABC234", session.Room.Code)
	assert.Empty(t, setup.publisher.events)
}

func TestCreateRetriesTakenCode(t *testing.T) {
	setup := newFixture("AAAAAA", "BBBBBB")
	setup.repository.createErrors = []error{model.ErrCodeTaken, nil}

	session, err := setup.service.Create(context.Background(), "Ona")

	require.NoError(t, err)
	assert.Equal(t, "BBBBBB", session.Room.Code)
	assert.Len(t, setup.repository.created, 2)
}

func TestCreateGivesUpAfterRepeatedCollisions(t *testing.T) {
	setup := newFixture()
	setup.repository.createErrors = []error{
		model.ErrCodeTaken, model.ErrCodeTaken, model.ErrCodeTaken, model.ErrCodeTaken, model.ErrCodeTaken,
	}

	_, err := setup.service.Create(context.Background(), "Ona")

	require.ErrorIs(t, err, model.ErrCodeTaken)
	assert.Len(t, setup.repository.created, 5)
}

func TestCreateRejectsInvalidName(t *testing.T) {
	setup := newFixture()

	_, err := setup.service.Create(context.Background(), "   ")

	require.ErrorIs(t, err, model.ErrInvalidDisplayName)
	assert.Empty(t, setup.repository.created)
}

func TestJoinPublishesParticipant(t *testing.T) {
	setup := newFixture()
	setup.repository.loaded = model.Room{
		Code:         "ABC234",
		Participants: []model.Participant{{ID: "host", IsHost: true}, {ID: "id-1", DisplayName: "Jonas"}},
	}

	session, err := setup.service.Join(context.Background(), "abc234", "Jonas")

	require.NoError(t, err)
	require.Len(t, setup.repository.added, 1)
	assert.Equal(t, "ABC234", setup.repository.added[0].RoomCode)
	assert.False(t, setup.repository.added[0].IsHost)
	assert.Len(t, session.Room.Participants, 2)
	require.Len(t, setup.publisher.events, 1)
	assert.Equal(t, model.EventParticipantJoined, setup.publisher.events[0].Type)
}

func TestJoinFailures(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		addErr   error
		expected error
	}{
		{name: "malformed code", code: "nope", expected: model.ErrRoomNotFound},
		{name: "missing room", code: "ABC234", addErr: model.ErrRoomNotFound, expected: model.ErrRoomNotFound},
		{name: "full room", code: "ABC234", addErr: model.ErrRoomFull, expected: model.ErrRoomFull},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setup := newFixture()
			setup.repository.addErr = testCase.addErr

			_, err := setup.service.Join(context.Background(), testCase.code, "Jonas")

			require.ErrorIs(t, err, testCase.expected)
			assert.Empty(t, setup.publisher.events)
		})
	}
}

func TestGetBuildsViewerState(t *testing.T) {
	setup := newFixture()
	setup.repository.loaded = model.Room{
		Code:         "ABC234",
		Participants: []model.Participant{{ID: "guest"}},
		Round:        &model.Round{Votes: []model.Vote{{ParticipantID: "guest", Card: model.CardFive}}},
	}

	view, err := setup.service.Get(context.Background(), model.Participant{ID: "guest", RoomCode: "ABC234"})

	require.NoError(t, err)
	assert.Equal(t, model.CardFive, view.OwnVote)
	assert.Empty(t, view.Round.Votes)
}

func TestLeave(t *testing.T) {
	testCases := []struct {
		name           string
		departure      model.Departure
		expectedEvents []model.EventType
		expectedClosed int
	}{
		{
			name:           "guest leaves",
			expectedEvents: []model.EventType{model.EventParticipantLeft},
		},
		{
			name:           "host leaves and role moves",
			departure:      model.Departure{NewHostID: "guest"},
			expectedEvents: []model.EventType{model.EventParticipantLeft, model.EventHostChanged},
		},
		{
			name:           "last participant leaves",
			departure:      model.Departure{RoomDeleted: true},
			expectedEvents: []model.EventType{},
			expectedClosed: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setup := newFixture()
			setup.repository.departure = testCase.departure

			err := setup.service.Leave(context.Background(), model.Participant{ID: "host", RoomCode: "ABC234"})

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedEvents, eventTypes(setup.publisher.events))
			assert.Len(t, setup.publisher.closed, testCase.expectedClosed)
		})
	}
}

func TestLeaveStoreFailurePublishesNothing(t *testing.T) {
	setup := newFixture()
	setup.repository.removeErr = errStore

	err := setup.service.Leave(context.Background(), model.Participant{ID: "host", RoomCode: "ABC234"})

	require.ErrorIs(t, err, errStore)
	assert.Empty(t, setup.publisher.events)
	assert.Empty(t, setup.publisher.closed)
}

func TestCloseRequiresHost(t *testing.T) {
	setup := newFixture()

	err := setup.service.Close(context.Background(), model.Participant{ID: "guest", RoomCode: "ABC234"})

	require.ErrorIs(t, err, model.ErrNotHost)
	assert.Empty(t, setup.repository.deleted)
	assert.Empty(t, setup.publisher.closed)
}

func TestCloseDeletesAndDisconnects(t *testing.T) {
	setup := newFixture()

	err := setup.service.Close(context.Background(), model.Participant{ID: "host", RoomCode: "ABC234", IsHost: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"ABC234"}, setup.repository.deleted)
	assert.Equal(t, []model.CloseReason{model.CloseReasonClosed}, setup.publisher.closed)
}

func eventTypes(events []model.Event) []model.EventType {
	types := make([]model.EventType, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}

	return types
}
