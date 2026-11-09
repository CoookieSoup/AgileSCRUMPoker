//go:build integration

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/app"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/infra/mysql"
)

type client struct {
	t       *testing.T
	baseURL string
	token   string
}

func (caller client) send(method string, path string, body any) (int, []byte) {
	caller.t.Helper()

	var payload io.Reader = http.NoBody

	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(caller.t, err)

		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(context.Background(), method, caller.baseURL+"/api/v1"+path, payload)
	require.NoError(caller.t, err)

	request.Header.Set("Content-Type", "application/json")

	if caller.token != "" {
		request.Header.Set("Authorization", "Bearer "+caller.token)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(caller.t, err)

	defer func() {
		_ = response.Body.Close()
	}()

	content, err := io.ReadAll(response.Body)
	require.NoError(caller.t, err)

	return response.StatusCode, content
}

func (caller client) session(method string, path string, body any) generated.Session {
	caller.t.Helper()

	status, content := caller.send(method, path, body)
	require.Equal(caller.t, http.StatusCreated, status, string(content))

	var session generated.Session

	require.NoError(caller.t, json.Unmarshal(content, &session))

	return session
}

func openDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	connectionString := os.Getenv("TEST_DATABASE_DSN")
	require.NotEmpty(t, connectionString, "TEST_DATABASE_DSN must be set")

	database, err := mysql.Open(connectionString)
	require.NoError(t, err)
	require.NoError(t, mysql.Migrate(database))

	return database
}

func listen(t *testing.T, baseURL string, code string, token string) *websocket.Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	address := "ws" + strings.TrimPrefix(baseURL, "http") + "/api/v1/rooms/" + code + "/events"

	connection, _, err := websocket.Dial(ctx, address, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = connection.CloseNow()
	})

	require.NoError(t, wsjson.Write(ctx, connection, map[string]string{"type": "auth", "token": token}))
	assert.Equal(t, "ready", nextType(t, connection))

	return connection
}

func nextRaw(t *testing.T, connection *websocket.Conn) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, content, err := connection.Read(ctx)
	require.NoError(t, err)

	return content
}

func nextType(t *testing.T, connection *websocket.Conn) string {
	t.Helper()

	var envelope struct {
		Type string `json:"type"`
	}

	require.NoError(t, json.Unmarshal(nextRaw(t, connection), &envelope))

	return envelope.Type
}

func countRows(t *testing.T, database *gorm.DB, code string) int64 {
	t.Helper()

	var total int64

	for _, query := range []string{
		"SELECT COUNT(*) FROM rooms WHERE code = ?",
		"SELECT COUNT(*) FROM participants WHERE room_code = ?",
		"SELECT COUNT(*) FROM rounds WHERE room_code = ?",
	} {
		var count int64

		require.NoError(t, database.Raw(query, code).Scan(&count).Error)

		total += count
	}

	return total
}

func TestFullRoomFlow(t *testing.T) {
	database := openDatabase(t)
	application := app.New(app.Options{
		Database:       database,
		Logger:         slog.New(slog.DiscardHandler),
		AllowedOrigins: []string{"*"},
		OriginPatterns: []string{"*"},
	})
	server := httptest.NewServer(application.Handler)
	t.Cleanup(server.Close)

	anonymous := client{t: t, baseURL: server.URL}
	hostSession := anonymous.session(http.MethodPost, "/rooms", map[string]string{"displayName": "Ona"})
	code := hostSession.Room.Code
	guestSession := anonymous.session(http.MethodPost, "/rooms/"+code+"/participants",
		map[string]string{"displayName": "Jonas"})
	host := client{t: t, baseURL: server.URL, token: hostSession.SessionToken}
	guest := client{t: t, baseURL: server.URL, token: guestSession.SessionToken}
	stream := listen(t, server.URL, code, guestSession.SessionToken)

	status, _ := host.send(http.MethodPost, "/rooms/"+code+"/round",
		map[string]string{"label": "Login", "jiraKey": "ABC-1"})
	require.Equal(t, http.StatusCreated, status)

	status, _ = guest.send(http.MethodPut, "/rooms/"+code+"/round/vote", map[string]string{"card": "8"})
	require.Equal(t, http.StatusNoContent, status)

	status, _ = host.send(http.MethodPut, "/rooms/"+code+"/round/vote", map[string]string{"card": "3"})
	require.Equal(t, http.StatusNoContent, status)

	status, content := guest.send(http.MethodGet, "/rooms/"+code, nil)
	require.Equal(t, http.StatusOK, status)

	var hidden generated.RoomState

	require.NoError(t, json.Unmarshal(content, &hidden))
	assert.Empty(t, hidden.Round.Votes)
	assert.Equal(t, "8", *hidden.OwnVote)

	for _, expected := range []string{"round_started", "participant_voted", "participant_voted"} {
		received := string(nextRaw(t, stream))

		assert.Contains(t, received, `"type":"`+expected+`"`)
		assert.NotContains(t, received, `"card"`, "card leaked before reveal")
	}

	status, _ = guest.send(http.MethodPost, "/rooms/"+code+"/round/reveal", nil)
	require.Equal(t, http.StatusForbidden, status)

	status, content = host.send(http.MethodPost, "/rooms/"+code+"/round/reveal", nil)
	require.Equal(t, http.StatusOK, status)

	var revealed generated.RoomState

	require.NoError(t, json.Unmarshal(content, &revealed))
	assert.ElementsMatch(t, []generated.Vote{
		{ParticipantId: hostSession.ParticipantId, Card: "3"},
		{ParticipantId: guestSession.ParticipantId, Card: "8"},
	}, revealed.Round.Votes)
	assert.Equal(t, "votes_revealed", nextType(t, stream))

	status, _ = guest.send(http.MethodPut, "/rooms/"+code+"/round/vote", map[string]string{"card": "1"})
	require.Equal(t, http.StatusConflict, status)

	status, _ = host.send(http.MethodDelete, "/rooms/"+code, nil)
	require.Equal(t, http.StatusNoContent, status)
	assert.Equal(t, "room_closed", nextType(t, stream))

	status, _ = guest.send(http.MethodGet, "/rooms/"+code, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Zero(t, countRows(t, database, code))
}
