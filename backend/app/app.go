package app

import (
	"log/slog"
	"net/http"

	"gorm.io/gorm"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api"
	healthapi "github.com/CoookieSoup/AgileSCRUMPoker/backend/api/health"
	realtimeapi "github.com/CoookieSoup/AgileSCRUMPoker/backend/api/realtime"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/respond"
	roomapi "github.com/CoookieSoup/AgileSCRUMPoker/backend/api/room"
	roundapi "github.com/CoookieSoup/AgileSCRUMPoker/backend/api/round"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/clock"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/mysql"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/random"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/realtime"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/service/cleanup"
	roomservice "github.com/CoookieSoup/AgileSCRUMPoker/backend/service/room"
	roundservice "github.com/CoookieSoup/AgileSCRUMPoker/backend/service/round"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/service/session"
)

type Options struct {
	Database       *gorm.DB
	Logger         *slog.Logger
	AllowedOrigins []string
	OriginPatterns []string
}

type Application struct {
	Handler http.Handler
	Cleanup cleanup.Service
}

func New(options Options) Application {
	systemClock := clock.System{}
	generator := random.Generator{}
	broker := realtime.NewBroker()

	sessions := session.New(mysql.NewSessionStore(options.Database), systemClock)
	rooms := roomservice.New(mysql.NewRoomStore(options.Database), broker, generator, systemClock)
	rounds := roundservice.New(mysql.NewRoundStore(options.Database), broker, generator, systemClock)
	sweeper := cleanup.New(mysql.NewCleanupStore(options.Database), broker, systemClock, options.Logger)

	handler := api.NewRouter(api.RouterOptions{
		Server: api.Server{
			RoomHandlers:   roomapi.New(rooms),
			RoundHandlers:  roundapi.New(rounds),
			HealthHandlers: healthapi.New(mysql.NewHealth(options.Database)),
		},
		Events:         realtimeapi.New(sessions, broker, options.OriginPatterns),
		Authenticator:  sessions,
		ErrorWriter:    respond.NewErrorWriter(options.Logger),
		AllowedOrigins: options.AllowedOrigins,
	})

	return Application{Handler: handler, Cleanup: sweeper}
}
