//go:build integration

package mysql_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/mysql"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/random"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

var (
	database  *gorm.DB
	generator random.Generator
)

func TestMain(main *testing.M) {
	connectionString := os.Getenv("TEST_DATABASE_DSN")
	if connectionString == "" {
		panic("TEST_DATABASE_DSN must be set for integration tests")
	}

	opened, err := mysql.Open(connectionString)
	if err != nil {
		panic(err)
	}

	err = mysql.Migrate(opened)
	if err != nil {
		panic(err)
	}

	database = opened

	os.Exit(main.Run())
}

func currentTime() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

func newParticipant(t *testing.T, code string, name string, isHost bool, joinedAt time.Time) model.Participant {
	t.Helper()

	participantID, err := generator.NewID()
	require.NoError(t, err)

	token, err := generator.NewToken()
	require.NoError(t, err)

	return model.Participant{
		ID:          participantID,
		RoomCode:    code,
		DisplayName: name,
		TokenHash:   model.HashToken(token),
		IsHost:      isHost,
		CreatedAt:   joinedAt,
	}
}

func createRoom(t *testing.T, expiresAt time.Time) (model.Room, model.Participant) {
	t.Helper()

	code, err := generator.NewCode()
	require.NoError(t, err)

	host := newParticipant(t, code, "Ona", true, currentTime())
	room := model.Room{
		Code:         code,
		ExpiresAt:    expiresAt,
		CreatedAt:    currentTime(),
		Participants: []model.Participant{host},
	}

	require.NoError(t, mysql.NewRoomStore(database).Create(context.Background(), room))

	t.Cleanup(func() {
		database.Where("code = ?", code).Delete(&model.Room{})
	})

	return room, host
}

func joinRoom(t *testing.T, code string, name string, joinedAt time.Time) model.Participant {
	t.Helper()

	participant := newParticipant(t, code, name, false, joinedAt)

	require.NoError(t, mysql.NewRoomStore(database).AddParticipant(context.Background(), participant, currentTime()))

	return participant
}

func countRows(t *testing.T, code string) int64 {
	t.Helper()

	var total int64

	for _, query := range []string{
		"SELECT COUNT(*) FROM rooms WHERE code = ?",
		"SELECT COUNT(*) FROM participants WHERE room_code = ?",
		"SELECT COUNT(*) FROM rounds WHERE room_code = ?",
		"SELECT COUNT(*) FROM votes JOIN rounds ON rounds.id = votes.round_id WHERE rounds.room_code = ?",
	} {
		var count int64

		require.NoError(t, database.Raw(query, code).Scan(&count).Error)

		total += count
	}

	return total
}

func TestCreateAndLoadRoom(t *testing.T) {
	room, host := createRoom(t, currentTime().Add(time.Hour))
	store := mysql.NewRoomStore(database)

	loaded, err := store.Load(context.Background(), room.Code, currentTime())

	require.NoError(t, err)
	assert.Equal(t, room.Code, loaded.Code)
	require.Len(t, loaded.Participants, 1)
	assert.Equal(t, host.ID, loaded.Participants[0].ID)
	assert.True(t, loaded.Participants[0].IsHost)
	assert.Nil(t, loaded.Round)
}

func TestCreateRejectsTakenCode(t *testing.T) {
	room, _ := createRoom(t, currentTime().Add(time.Hour))
	duplicate := model.Room{Code: room.Code, ExpiresAt: room.ExpiresAt, CreatedAt: room.CreatedAt}

	err := mysql.NewRoomStore(database).Create(context.Background(), duplicate)

	require.ErrorIs(t, err, model.ErrCodeTaken)
}

func TestExpiredRoomIsHiddenAndClosedToJoins(t *testing.T) {
	room, _ := createRoom(t, currentTime().Add(-time.Second))
	store := mysql.NewRoomStore(database)

	_, err := store.Load(context.Background(), room.Code, currentTime())

	require.ErrorIs(t, err, model.ErrRoomNotFound)

	err = store.AddParticipant(context.Background(), newParticipant(t, room.Code, "Jonas", false, currentTime()),
		currentTime())

	require.ErrorIs(t, err, model.ErrRoomNotFound)
}

func TestConcurrentJoinsRespectLimit(t *testing.T) {
	room, _ := createRoom(t, currentTime().Add(time.Hour))
	store := mysql.NewRoomStore(database)

	var (
		group  sync.WaitGroup
		mutex  sync.Mutex
		joined int
		full   int
	)

	for range 25 {
		participant := newParticipant(t, room.Code, "Guest", false, currentTime())

		group.Go(func() {
			err := store.AddParticipant(context.Background(), participant, currentTime())

			mutex.Lock()
			defer mutex.Unlock()

			switch {
			case err == nil:
				joined++
			case errors.Is(err, model.ErrRoomFull):
				full++
			default:
				t.Errorf("unexpected join error: %v", err)
			}
		})
	}

	group.Wait()

	loaded, err := store.Load(context.Background(), room.Code, currentTime())

	require.NoError(t, err)
	assert.Equal(t, 19, joined)
	assert.Equal(t, 6, full)
	assert.Len(t, loaded.Participants, model.MaxParticipants)
}

func TestVotingLifecycle(t *testing.T) {
	room, host := createRoom(t, currentTime().Add(time.Hour))
	guest := joinRoom(t, room.Code, "Jonas", currentTime())
	rounds := mysql.NewRoundStore(database)
	ctx := context.Background()

	err := rounds.UpsertVote(ctx, room.Code, guest.ID, model.CardFive)

	require.ErrorIs(t, err, model.ErrRoundNotFound)
	require.NoError(t, rounds.Replace(ctx, model.Round{ID: "round-a-" + room.Code, RoomCode: room.Code, Label: "A"}))
	require.NoError(t, rounds.UpsertVote(ctx, room.Code, guest.ID, model.CardFive))
	require.NoError(t, rounds.UpsertVote(ctx, room.Code, guest.ID, model.CardEight))
	require.NoError(t, rounds.UpsertVote(ctx, room.Code, host.ID, model.CardThree))

	loaded, err := rounds.Load(ctx, room.Code, currentTime())

	require.NoError(t, err)
	require.NotNil(t, loaded.Round)
	assert.ElementsMatch(t, []model.Vote{
		{RoundID: "round-a-" + room.Code, ParticipantID: guest.ID, Card: model.CardEight},
		{RoundID: "round-a-" + room.Code, ParticipantID: host.ID, Card: model.CardThree},
	}, loaded.Round.Votes)

	require.NoError(t, rounds.Reveal(ctx, room.Code))
	require.NoError(t, rounds.Reveal(ctx, room.Code))

	err = rounds.UpsertVote(ctx, room.Code, guest.ID, model.CardOne)

	require.ErrorIs(t, err, model.ErrRoundRevealed)
	require.NoError(t, rounds.Replace(ctx, model.Round{ID: "round-b-" + room.Code, RoomCode: room.Code, Label: "B"}))

	loaded, err = rounds.Load(ctx, room.Code, currentTime())

	require.NoError(t, err)
	assert.Equal(t, "B", loaded.Round.Label)
	assert.False(t, loaded.Round.Revealed)
	assert.Empty(t, loaded.Round.Votes)
}

func TestRevealWithoutRound(t *testing.T) {
	room, _ := createRoom(t, currentTime().Add(time.Hour))

	err := mysql.NewRoundStore(database).Reveal(context.Background(), room.Code)

	require.ErrorIs(t, err, model.ErrRoundNotFound)
}

func TestHostLeavingPromotesEarliestParticipant(t *testing.T) {
	room, host := createRoom(t, currentTime().Add(time.Hour))
	first := joinRoom(t, room.Code, "Jonas", currentTime().Add(time.Second))
	joinRoom(t, room.Code, "Rūta", currentTime().Add(2*time.Second))

	store := mysql.NewRoomStore(database)

	departure, err := store.RemoveParticipant(context.Background(), room.Code, host.ID)

	require.NoError(t, err)
	assert.Equal(t, model.Departure{NewHostID: first.ID}, departure)

	loaded, err := store.Load(context.Background(), room.Code, currentTime())

	require.NoError(t, err)
	require.Len(t, loaded.Participants, 2)
	assert.True(t, loaded.Participants[0].IsHost)
	assert.False(t, loaded.Participants[1].IsHost)
}

func TestGuestLeavingKeepsHost(t *testing.T) {
	room, _ := createRoom(t, currentTime().Add(time.Hour))
	guest := joinRoom(t, room.Code, "Jonas", currentTime())

	departure, err := mysql.NewRoomStore(database).RemoveParticipant(context.Background(), room.Code, guest.ID)

	require.NoError(t, err)
	assert.Equal(t, model.Departure{}, departure)
}

func TestLastParticipantLeavingDeletesEverything(t *testing.T) {
	room, host := createRoom(t, currentTime().Add(time.Hour))
	rounds := mysql.NewRoundStore(database)
	ctx := context.Background()

	require.NoError(t, rounds.Replace(ctx, model.Round{ID: "round-" + room.Code, RoomCode: room.Code}))
	require.NoError(t, rounds.UpsertVote(ctx, room.Code, host.ID, model.CardTwo))

	departure, err := mysql.NewRoomStore(database).RemoveParticipant(ctx, room.Code, host.ID)

	require.NoError(t, err)
	assert.True(t, departure.RoomDeleted)
	assert.Zero(t, countRows(t, room.Code))
}

func TestDeleteCascades(t *testing.T) {
	room, host := createRoom(t, currentTime().Add(time.Hour))
	guest := joinRoom(t, room.Code, "Jonas", currentTime())
	rounds := mysql.NewRoundStore(database)
	ctx := context.Background()

	require.NoError(t, rounds.Replace(ctx, model.Round{ID: "round-" + room.Code, RoomCode: room.Code}))
	require.NoError(t, rounds.UpsertVote(ctx, room.Code, host.ID, model.CardTwo))
	require.NoError(t, rounds.UpsertVote(ctx, room.Code, guest.ID, model.CardThree))
	assert.Equal(t, int64(6), countRows(t, room.Code))
	require.NoError(t, mysql.NewRoomStore(database).Delete(ctx, room.Code))
	assert.Zero(t, countRows(t, room.Code))
	require.ErrorIs(t, mysql.NewRoomStore(database).Delete(ctx, room.Code), model.ErrRoomNotFound)
}

func TestFindByTokenHash(t *testing.T) {
	room, host := createRoom(t, currentTime().Add(time.Hour))
	sessions := mysql.NewSessionStore(database)

	participant, found, err := sessions.FindByTokenHash(context.Background(), host.TokenHash)

	require.NoError(t, err)
	assert.Equal(t, host.ID, participant.ID)
	assert.Equal(t, room.Code, found.Code)

	_, _, err = sessions.FindByTokenHash(context.Background(), model.HashToken("unknown"))

	require.ErrorIs(t, err, model.ErrUnauthorized)
}

func TestDeleteExpiredRemovesOnlyExpiredRooms(t *testing.T) {
	expired, _ := createRoom(t, currentTime().Add(-time.Minute))
	active, _ := createRoom(t, currentTime().Add(time.Hour))

	codes, err := mysql.NewCleanupStore(database).DeleteExpired(context.Background(), currentTime())

	require.NoError(t, err)
	assert.Contains(t, codes, expired.Code)
	assert.NotContains(t, codes, active.Code)
	assert.Zero(t, countRows(t, expired.Code))
	assert.Equal(t, int64(2), countRows(t, active.Code))
}

func TestHealthPing(t *testing.T) {
	require.NoError(t, mysql.NewHealth(database).Ping(context.Background()))
}
