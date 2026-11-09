package room

import (
	"context"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/api/middleware"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/api/presenter"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type Service interface {
	Create(ctx context.Context, displayName string) (model.Session, error)
	Join(ctx context.Context, code string, displayName string) (model.Session, error)
	Get(ctx context.Context, actor model.Participant) (model.RoomView, error)
	Leave(ctx context.Context, actor model.Participant) error
	Close(ctx context.Context, actor model.Participant) error
}

type Controller struct {
	service Service
}

func New(service Service) Controller {
	return Controller{service: service}
}

func (controller Controller) CreateRoom(
	ctx context.Context,
	request generated.CreateRoomRequestObject,
) (generated.CreateRoomResponseObject, error) {
	session, err := controller.service.Create(ctx, request.Body.DisplayName)
	if err != nil {
		return nil, err
	}

	return generated.CreateRoom201JSONResponse(presenter.Session(session)), nil
}

func (controller Controller) JoinRoom(
	ctx context.Context,
	request generated.JoinRoomRequestObject,
) (generated.JoinRoomResponseObject, error) {
	session, err := controller.service.Join(ctx, request.Code, request.Body.DisplayName)
	if err != nil {
		return nil, err
	}

	return generated.JoinRoom201JSONResponse(presenter.Session(session)), nil
}

func (controller Controller) GetRoom(
	ctx context.Context,
	_ generated.GetRoomRequestObject,
) (generated.GetRoomResponseObject, error) {
	actor, err := middleware.Participant(ctx)
	if err != nil {
		return nil, err
	}

	view, err := controller.service.Get(ctx, actor)
	if err != nil {
		return nil, err
	}

	return generated.GetRoom200JSONResponse(presenter.RoomState(view)), nil
}

func (controller Controller) LeaveRoom(
	ctx context.Context,
	_ generated.LeaveRoomRequestObject,
) (generated.LeaveRoomResponseObject, error) {
	actor, err := middleware.Participant(ctx)
	if err != nil {
		return nil, err
	}

	err = controller.service.Leave(ctx, actor)
	if err != nil {
		return nil, err
	}

	return generated.LeaveRoom204Response{}, nil
}

func (controller Controller) CloseRoom(
	ctx context.Context,
	_ generated.CloseRoomRequestObject,
) (generated.CloseRoomResponseObject, error) {
	actor, err := middleware.Participant(ctx)
	if err != nil {
		return nil, err
	}

	err = controller.service.Close(ctx, actor)
	if err != nil {
		return nil, err
	}

	return generated.CloseRoom204Response{}, nil
}
