package realtime

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

const (
	authTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	pingInterval = 30 * time.Second

	StatusUnauthorized websocket.StatusCode = 4401
	StatusRoomNotFound websocket.StatusCode = 4404
)

type Authenticator interface {
	Authenticate(ctx context.Context, code string, token string) (model.Participant, error)
}

type Subscriber interface {
	Subscribe(code string) (<-chan model.Event, func())
}

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type readyMessage struct {
	Type string `json:"type"`
}

type Controller struct {
	authenticator  Authenticator
	subscriber     Subscriber
	originPatterns []string
}

func New(authenticator Authenticator, subscriber Subscriber, originPatterns []string) Controller {
	return Controller{
		authenticator:  authenticator,
		subscriber:     subscriber,
		originPatterns: originPatterns,
	}
}

func (controller Controller) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	connection, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		OriginPatterns: controller.originPatterns,
	})
	if err != nil {
		return
	}

	defer func() {
		_ = connection.CloseNow()
	}()

	participant, err := controller.authenticate(request.Context(), connection, chi.URLParam(request, "code"))
	if err != nil {
		_ = connection.Close(closeStatus(err), closeReason(err))

		return
	}

	events, unsubscribe := controller.subscriber.Subscribe(participant.RoomCode)
	defer unsubscribe()

	err = write(request.Context(), connection, readyMessage{Type: "ready"})
	if err != nil {
		return
	}

	ctx := connection.CloseRead(request.Context())
	controller.stream(ctx, connection, events)
}

func (controller Controller) authenticate(
	ctx context.Context,
	connection *websocket.Conn,
	code string,
) (model.Participant, error) {
	readContext, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()

	var message authMessage

	err := wsjson.Read(readContext, connection, &message)
	if err != nil || message.Type != "auth" {
		return model.Participant{}, model.ErrUnauthorized
	}

	return controller.authenticator.Authenticate(ctx, code, message.Token)
}

func (Controller) stream(ctx context.Context, connection *websocket.Conn, events <-chan model.Event) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := ping(ctx, connection)
			if err != nil {
				return
			}
		case event, open := <-events:
			if !open {
				_ = connection.Close(websocket.StatusNormalClosure, "stream ended")

				return
			}

			err := write(ctx, connection, event)
			if err != nil {
				return
			}
		}
	}
}

func write(ctx context.Context, connection *websocket.Conn, message any) error {
	writeContext, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	return wsjson.Write(writeContext, connection, message)
}

func ping(ctx context.Context, connection *websocket.Conn) error {
	pingContext, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	return connection.Ping(pingContext)
}

func closeStatus(err error) websocket.StatusCode {
	if errors.Is(err, model.ErrRoomNotFound) {
		return StatusRoomNotFound
	}

	return StatusUnauthorized
}

func closeReason(err error) string {
	if errors.Is(err, model.ErrRoomNotFound) {
		return "room_not_found"
	}

	return "unauthorized"
}
