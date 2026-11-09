package respond

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type Logger interface {
	ErrorContext(ctx context.Context, message string, args ...any)
}

type problem struct {
	target error
	status int
	code   string
}

var problems = []problem{
	{target: model.ErrRoomNotFound, status: http.StatusNotFound, code: "room_not_found"},
	{target: model.ErrRoundNotFound, status: http.StatusNotFound, code: "round_not_found"},
	{target: model.ErrUnauthorized, status: http.StatusUnauthorized, code: "unauthorized"},
	{target: model.ErrNotHost, status: http.StatusForbidden, code: "not_host"},
	{target: model.ErrRoomFull, status: http.StatusConflict, code: "room_full"},
	{target: model.ErrRoundRevealed, status: http.StatusConflict, code: "round_revealed"},
	{target: model.ErrInvalidDisplayName, status: http.StatusUnprocessableEntity, code: "invalid_display_name"},
	{target: model.ErrInvalidCard, status: http.StatusUnprocessableEntity, code: "invalid_card"},
	{target: model.ErrInvalidLabel, status: http.StatusUnprocessableEntity, code: "invalid_label"},
	{target: model.ErrInvalidJiraKey, status: http.StatusUnprocessableEntity, code: "invalid_jira_key"},
	{target: model.ErrUnavailable, status: http.StatusServiceUnavailable, code: "unavailable"},
}

type ErrorWriter struct {
	logger Logger
}

func NewErrorWriter(logger Logger) ErrorWriter {
	return ErrorWriter{logger: logger}
}

func (errorWriter ErrorWriter) Write(writer http.ResponseWriter, request *http.Request, err error) {
	for _, entry := range problems {
		if errors.Is(err, entry.target) {
			writeError(writer, entry.status, entry.code, entry.target.Error())

			return
		}
	}

	errorWriter.logger.ErrorContext(request.Context(), "request failed", "error", err)
	writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
}

func (ErrorWriter) BadRequest(writer http.ResponseWriter, _ *http.Request, err error) {
	writeError(writer, http.StatusBadRequest, "invalid_request", err.Error())
}

func writeError(writer http.ResponseWriter, status int, code string, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	_ = json.NewEncoder(writer).Encode(generated.Error{Code: code, Message: message})
}
