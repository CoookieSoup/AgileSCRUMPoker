package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type Authenticator interface {
	Authenticate(ctx context.Context, code string, token string) (model.Participant, error)
}

type ErrorHandler func(writer http.ResponseWriter, request *http.Request, err error)

type participantKey struct{}

func Authentication(authenticator Authenticator, onError ErrorHandler) generated.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, secured := request.Context().Value(generated.BearerAuthScopes).([]string)
			if !secured {
				next.ServeHTTP(writer, request)

				return
			}

			token, _ := strings.CutPrefix(request.Header.Get("Authorization"), "Bearer ")

			participant, err := authenticator.Authenticate(
				request.Context(),
				chi.URLParam(request, "code"),
				strings.TrimSpace(token),
			)
			if err != nil {
				onError(writer, request, err)

				return
			}

			ctx := context.WithValue(request.Context(), participantKey{}, participant)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func Participant(ctx context.Context) (model.Participant, error) {
	participant, found := ctx.Value(participantKey{}).(model.Participant)
	if !found {
		return model.Participant{}, model.ErrUnauthorized
	}

	return participant, nil
}
