package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/health"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/respond"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/room"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/round"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

var (
	errBroken = errors.New("broken")
	view      = model.RoomView{
		Code:         "ABC234",
		ExpiresAt:    time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC),
		Participants: []model.ParticipantView{{ID: "host", DisplayName: "Ona", IsHost: true}},
	}
	session = model.Session{ParticipantID: "host", Token: "secret", Room: view}
)

type fakeRoomService struct {
	err      error
	actor    model.Participant
	joinCode string
}

func (service *fakeRoomService) Create(_ context.Context, _ string) (model.Session, error) {
	return session, service.err
}

func (service *fakeRoomService) Join(_ context.Context, code string, _ string) (model.Session, error) {
	service.joinCode = code

	return session, service.err
}

func (service *fakeRoomService) Get(_ context.Context, actor model.Participant) (model.RoomView, error) {
	service.actor = actor

	return view, service.err
}

func (service *fakeRoomService) Leave(_ context.Context, actor model.Participant) error {
	service.actor = actor

	return service.err
}

func (service *fakeRoomService) Close(_ context.Context, actor model.Participant) error {
	service.actor = actor

	return service.err
}

type fakeRoundService struct {
	err     error
	label   string
	jiraKey string
	card    string
}

func (service *fakeRoundService) Start(
	_ context.Context,
	_ model.Participant,
	label string,
	jiraKey string,
) (model.RoomView, error) {
	service.label = label
	service.jiraKey = jiraKey

	return view, service.err
}

func (service *fakeRoundService) Vote(_ context.Context, _ model.Participant, card string) error {
	service.card = card

	return service.err
}

func (service *fakeRoundService) Reveal(_ context.Context, _ model.Participant) (model.RoomView, error) {
	return view, service.err
}

type fakePinger struct {
	err error
}

func (pinger fakePinger) Ping(_ context.Context) error {
	return pinger.err
}

type fakeAuthenticator struct{}

func (fakeAuthenticator) Authenticate(_ context.Context, code string, token string) (model.Participant, error) {
	if token != "host-token" {
		return model.Participant{}, model.ErrUnauthorized
	}

	return model.Participant{ID: "host", RoomCode: code, IsHost: true}, nil
}

type fixture struct {
	rooms  *fakeRoomService
	rounds *fakeRoundService
	router http.Handler
}

func newFixture(pingErr error) fixture {
	rooms := &fakeRoomService{}
	rounds := &fakeRoundService{}

	router := api.NewRouter(api.RouterOptions{
		Server: api.Server{
			RoomHandlers:   room.New(rooms),
			RoundHandlers:  round.New(rounds),
			HealthHandlers: health.New(fakePinger{err: pingErr}),
		},
		Events:         http.NotFoundHandler(),
		Authenticator:  fakeAuthenticator{},
		ErrorWriter:    respond.NewErrorWriter(slog.New(slog.DiscardHandler)),
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	return fixture{rooms: rooms, rounds: rounds, router: router}
}

func (setup fixture) call(method string, path string, token string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	setup.router.ServeHTTP(recorder, request)

	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var body generated.Error

	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))

	return body.Code
}

type endpoint struct {
	method string
	path   string
	body   string
	status int
}

var endpoints = []endpoint{
	{method: http.MethodPost, path: "/rooms", body: `{"displayName":"Ona"}`, status: http.StatusCreated},
	{
		method: http.MethodPost,
		path:   "/rooms/ABC234/participants",
		body:   `{"displayName":"Jo"}`,
		status: http.StatusCreated,
	},
	{method: http.MethodGet, path: "/rooms/ABC234", status: http.StatusOK},
	{method: http.MethodDelete, path: "/rooms/ABC234", status: http.StatusNoContent},
	{method: http.MethodDelete, path: "/rooms/ABC234/participants/me", status: http.StatusNoContent},
	{method: http.MethodPost, path: "/rooms/ABC234/round", body: `{"label":"Login"}`, status: http.StatusCreated},
	{method: http.MethodPut, path: "/rooms/ABC234/round/vote", body: `{"card":"5"}`, status: http.StatusNoContent},
	{method: http.MethodPost, path: "/rooms/ABC234/round/reveal", status: http.StatusOK},
}

func isPublic(target endpoint) bool {
	return target.method == http.MethodPost &&
		(target.path == "/rooms" || strings.HasSuffix(target.path, "/participants"))
}

func TestEndpointsSucceed(t *testing.T) {
	for _, target := range endpoints {
		t.Run(target.method+" "+target.path, func(t *testing.T) {
			recorder := newFixture(nil).call(target.method, target.path, "host-token", target.body)

			assert.Equal(t, target.status, recorder.Code, recorder.Body.String())
		})
	}
}

func TestSecuredEndpointsRequireToken(t *testing.T) {
	for _, target := range endpoints {
		t.Run(target.method+" "+target.path, func(t *testing.T) {
			recorder := newFixture(nil).call(target.method, target.path, "wrong-token", target.body)

			if isPublic(target) {
				assert.Equal(t, target.status, recorder.Code)

				return
			}

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assert.Equal(t, "unauthorized", errorCode(t, recorder))
		})
	}
}

func TestEndpointsMapServiceErrors(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not found", err: model.ErrRoomNotFound, status: http.StatusNotFound, code: "room_not_found"},
		{name: "not host", err: model.ErrNotHost, status: http.StatusForbidden, code: "not_host"},
		{name: "unexpected", err: errBroken, status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, target := range endpoints {
		for _, testCase := range testCases {
			t.Run(target.method+" "+target.path+" "+testCase.name, func(t *testing.T) {
				setup := newFixture(nil)
				setup.rooms.err = testCase.err
				setup.rounds.err = testCase.err

				recorder := setup.call(target.method, target.path, "host-token", target.body)

				assert.Equal(t, testCase.status, recorder.Code)
				assert.Equal(t, testCase.code, errorCode(t, recorder))
			})
		}
	}
}

func TestCreateReturnsSession(t *testing.T) {
	recorder := newFixture(nil).call(http.MethodPost, "/rooms", "", `{"displayName":"Ona"}`)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.JSONEq(t, `{
		"participantId": "host",
		"sessionToken": "secret",
		"room": {
			"code": "ABC234",
			"expiresAt": "2026-01-01T13:00:00Z",
			"participants": [{"id": "host", "displayName": "Ona", "isHost": true, "hasVoted": false}]
		}
	}`, recorder.Body.String())
}

func TestRequestsReachServicesWithInput(t *testing.T) {
	setup := newFixture(nil)

	setup.call(http.MethodPost, "/rooms/abc234/participants", "", `{"displayName":"Jo"}`)
	setup.call(http.MethodPost, "/rooms/ABC234/round", "host-token", `{"label":"Login","jiraKey":"ABC-1"}`)
	setup.call(http.MethodPut, "/rooms/ABC234/round/vote", "host-token", `{"card":"13"}`)
	setup.call(http.MethodGet, "/rooms/ABC234", "host-token", "")

	assert.Equal(t, "abc234", setup.rooms.joinCode)
	assert.Equal(t, "Login", setup.rounds.label)
	assert.Equal(t, "ABC-1", setup.rounds.jiraKey)
	assert.Equal(t, "13", setup.rounds.card)
	assert.Equal(t, model.Participant{ID: "host", RoomCode: "ABC234", IsHost: true}, setup.rooms.actor)
}

func TestMalformedBodyIsBadRequest(t *testing.T) {
	recorder := newFixture(nil).call(http.MethodPost, "/rooms", "", `{"displayName":`)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "invalid_request", errorCode(t, recorder))
}

func TestHealth(t *testing.T) {
	healthy := newFixture(nil).call(http.MethodGet, "/healthz", "", "")

	assert.Equal(t, http.StatusOK, healthy.Code)
	assert.JSONEq(t, `{"status":"ok"}`, healthy.Body.String())

	unhealthy := newFixture(errBroken).call(http.MethodGet, "/healthz", "", "")

	assert.Equal(t, http.StatusServiceUnavailable, unhealthy.Code)
	assert.Equal(t, "unavailable", errorCode(t, unhealthy))
}

func TestCorsPreflight(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/rooms", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "Content-Type")

	recorder := httptest.NewRecorder()
	newFixture(nil).router.ServeHTTP(recorder, request)

	assert.Equal(t, "http://localhost:5173", recorder.Header().Get("Access-Control-Allow-Origin"))
}
