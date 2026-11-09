package respond_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/respond"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type fakeLogger struct {
	messages []string
}

func (logger *fakeLogger) ErrorContext(_ context.Context, message string, _ ...any) {
	logger.messages = append(logger.messages, message)
}

func TestWriteMapsDomainErrors(t *testing.T) {
	testCases := []struct {
		err    error
		status int
		code   string
	}{
		{err: model.ErrRoomNotFound, status: http.StatusNotFound, code: "room_not_found"},
		{err: fmt.Errorf("load room: %w", model.ErrRoomNotFound), status: http.StatusNotFound, code: "room_not_found"},
		{err: model.ErrRoundNotFound, status: http.StatusNotFound, code: "round_not_found"},
		{err: model.ErrUnauthorized, status: http.StatusUnauthorized, code: "unauthorized"},
		{err: model.ErrNotHost, status: http.StatusForbidden, code: "not_host"},
		{err: model.ErrRoomFull, status: http.StatusConflict, code: "room_full"},
		{err: model.ErrRoundRevealed, status: http.StatusConflict, code: "round_revealed"},
		{err: model.ErrInvalidDisplayName, status: http.StatusUnprocessableEntity, code: "invalid_display_name"},
		{err: model.ErrInvalidCard, status: http.StatusUnprocessableEntity, code: "invalid_card"},
		{err: model.ErrInvalidLabel, status: http.StatusUnprocessableEntity, code: "invalid_label"},
		{err: model.ErrInvalidJiraKey, status: http.StatusUnprocessableEntity, code: "invalid_jira_key"},
		{err: model.ErrUnavailable, status: http.StatusServiceUnavailable, code: "unavailable"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.code, func(t *testing.T) {
			logger := &fakeLogger{}
			recorder := httptest.NewRecorder()

			respond.NewErrorWriter(logger).Write(recorder, httptest.NewRequest(http.MethodGet, "/", nil), testCase.err)

			assert.Equal(t, testCase.status, recorder.Code)
			assert.Equal(t, testCase.code, decode(t, recorder).Code)
			assert.Empty(t, logger.messages)
		})
	}
}

func TestWriteHidesAndLogsUnexpectedErrors(t *testing.T) {
	logger := &fakeLogger{}
	recorder := httptest.NewRecorder()

	respond.NewErrorWriter(logger).Write(recorder, httptest.NewRequest(http.MethodGet, "/", nil),
		errors.New("connection refused to 10.0.0.1"))

	body := decode(t, recorder)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "internal_error", body.Code)
	assert.NotContains(t, body.Message, "10.0.0.1")
	assert.Len(t, logger.messages, 1)
}

func TestBadRequest(t *testing.T) {
	recorder := httptest.NewRecorder()

	respond.NewErrorWriter(&fakeLogger{}).BadRequest(recorder, httptest.NewRequest(http.MethodGet, "/", nil),
		errors.New("bad json"))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "invalid_request", decode(t, recorder).Code)
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) generated.Error {
	t.Helper()

	var body generated.Error

	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))

	return body
}
