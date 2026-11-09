package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/middleware"
)

const corsMaxAge = 300

type RoomHandlers interface {
	CreateRoom(ctx context.Context, request generated.CreateRoomRequestObject) (generated.CreateRoomResponseObject, error)
	JoinRoom(ctx context.Context, request generated.JoinRoomRequestObject) (generated.JoinRoomResponseObject, error)
	GetRoom(ctx context.Context, request generated.GetRoomRequestObject) (generated.GetRoomResponseObject, error)
	LeaveRoom(ctx context.Context, request generated.LeaveRoomRequestObject) (generated.LeaveRoomResponseObject, error)
	CloseRoom(ctx context.Context, request generated.CloseRoomRequestObject) (generated.CloseRoomResponseObject, error)
}

type RoundHandlers interface {
	StartRound(
		ctx context.Context,
		request generated.StartRoundRequestObject,
	) (generated.StartRoundResponseObject, error)
	CastVote(ctx context.Context, request generated.CastVoteRequestObject) (generated.CastVoteResponseObject, error)
	RevealVotes(
		ctx context.Context,
		request generated.RevealVotesRequestObject,
	) (generated.RevealVotesResponseObject, error)
}

type HealthHandlers interface {
	CheckHealth(
		ctx context.Context,
		request generated.CheckHealthRequestObject,
	) (generated.CheckHealthResponseObject, error)
}

type ErrorWriter interface {
	Write(writer http.ResponseWriter, request *http.Request, err error)
	BadRequest(writer http.ResponseWriter, request *http.Request, err error)
}

type Server struct {
	RoomHandlers
	RoundHandlers
	HealthHandlers
}

type RouterOptions struct {
	Server         Server
	Events         http.Handler
	Authenticator  middleware.Authenticator
	ErrorWriter    ErrorWriter
	AllowedOrigins []string
}

func NewRouter(options RouterOptions) http.Handler {
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: options.AllowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         corsMaxAge,
	}))

	router.Get("/api/v1/rooms/{code}/events", options.Events.ServeHTTP)

	strictHandler := generated.NewStrictHandlerWithOptions(options.Server, nil, generated.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  options.ErrorWriter.BadRequest,
		ResponseErrorHandlerFunc: options.ErrorWriter.Write,
	})

	generated.HandlerWithOptions(strictHandler, generated.ChiServerOptions{
		BaseURL:    "/api/v1",
		BaseRouter: router,
		Middlewares: []generated.MiddlewareFunc{
			middleware.Authentication(options.Authenticator, options.ErrorWriter.Write),
		},
		ErrorHandlerFunc: options.ErrorWriter.BadRequest,
	})

	return router
}
