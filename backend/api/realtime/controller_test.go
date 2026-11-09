package realtime_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/realtime"
	infrarealtime "github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/realtime"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type fakeAuthenticator struct{}

func (fakeAuthenticator) Authenticate(_ context.Context, code string, token string) (model.Participant, error) {
	switch token {
	case "good-token":
		return model.Participant{ID: "guest", RoomCode: code}, nil
	case "expired-token":
		return model.Participant{}, model.ErrRoomNotFound
	default:
		return model.Participant{}, model.ErrUnauthorized
	}
}

type message struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

func startServer(t *testing.T) (*infrarealtime.Broker, string) {
	t.Helper()

	broker := infrarealtime.NewBroker()
	router := chi.NewRouter()
	router.Get("/rooms/{code}/events", realtime.New(fakeAuthenticator{}, broker, []string{"*"}).ServeHTTP)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return broker, "ws" + strings.TrimPrefix(server.URL, "http") + "/rooms/ABC234/events"
}

func connect(t *testing.T, address string, token string) *websocket.Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connection, _, err := websocket.Dial(ctx, address, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = connection.CloseNow()
	})

	require.NoError(t, wsjson.Write(ctx, connection, map[string]string{"type": "auth", "token": token}))

	return connection
}

func read(t *testing.T, connection *websocket.Conn) (message, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var received message

	err := wsjson.Read(ctx, connection, &received)

	return received, err
}

func TestStreamsRoomEventsAfterReady(t *testing.T) {
	broker, address := startServer(t)
	connection := connect(t, address, "good-token")

	ready, err := read(t, connection)

	require.NoError(t, err)
	assert.Equal(t, "ready", ready.Type)

	broker.Publish("ABC234", model.ParticipantVotedEvent("host"))
	broker.Publish("XYZ789", model.ParticipantVotedEvent("other"))
	broker.Publish("ABC234", model.RoundStartedEvent(model.Round{Label: "Login"}))

	voted, err := read(t, connection)

	require.NoError(t, err)
	assert.Equal(t, message{Type: "participant_voted", Payload: map[string]any{"participantId": "host"}}, voted)

	started, err := read(t, connection)

	require.NoError(t, err)
	assert.Equal(t, "round_started", started.Type)
}

func TestRoomCloseEndsStream(t *testing.T) {
	broker, address := startServer(t)
	connection := connect(t, address, "good-token")

	_, err := read(t, connection)
	require.NoError(t, err)

	broker.CloseRoom("ABC234", model.CloseReasonClosed)

	closed, err := read(t, connection)

	require.NoError(t, err)
	assert.Equal(t, message{Type: "room_closed", Payload: map[string]any{"reason": "closed"}}, closed)

	_, err = read(t, connection)

	assert.Equal(t, websocket.StatusNormalClosure, websocket.CloseStatus(err))
}

func TestAuthenticationFailures(t *testing.T) {
	testCases := []struct {
		name   string
		token  string
		status websocket.StatusCode
	}{
		{name: "bad token", token: "bad-token", status: realtime.StatusUnauthorized},
		{name: "expired room", token: "expired-token", status: realtime.StatusRoomNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, address := startServer(t)
			connection := connect(t, address, testCase.token)

			_, err := read(t, connection)

			assert.Equal(t, testCase.status, websocket.CloseStatus(err))
		})
	}
}

func TestWrongFirstMessageIsRejected(t *testing.T) {
	_, address := startServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connection, _, err := websocket.Dial(ctx, address, nil)
	require.NoError(t, err)

	defer func() {
		_ = connection.CloseNow()
	}()

	require.NoError(t, wsjson.Write(ctx, connection, map[string]string{"type": "hello", "token": "good-token"}))

	_, err = read(t, connection)

	assert.Equal(t, realtime.StatusUnauthorized, websocket.CloseStatus(err))
}
