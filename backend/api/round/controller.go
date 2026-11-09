package round

import (
	"context"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/middleware"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/presenter"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

type Service interface {
	Start(ctx context.Context, actor model.Participant, label string, jiraKey string) (model.RoomView, error)
	Vote(ctx context.Context, actor model.Participant, card string) error
	Reveal(ctx context.Context, actor model.Participant) (model.RoomView, error)
}

type Controller struct {
	service Service
}

func New(service Service) Controller {
	return Controller{service: service}
}

func (controller Controller) StartRound(
	ctx context.Context,
	request generated.StartRoundRequestObject,
) (generated.StartRoundResponseObject, error) {
	actor, err := middleware.Participant(ctx)
	if err != nil {
		return nil, err
	}

	view, err := controller.service.Start(ctx, actor, valueOf(request.Body.Label), valueOf(request.Body.JiraKey))
	if err != nil {
		return nil, err
	}

	return generated.StartRound201JSONResponse(presenter.RoomState(view)), nil
}

func (controller Controller) CastVote(
	ctx context.Context,
	request generated.CastVoteRequestObject,
) (generated.CastVoteResponseObject, error) {
	actor, err := middleware.Participant(ctx)
	if err != nil {
		return nil, err
	}

	err = controller.service.Vote(ctx, actor, request.Body.Card)
	if err != nil {
		return nil, err
	}

	return generated.CastVote204Response{}, nil
}

func (controller Controller) RevealVotes(
	ctx context.Context,
	_ generated.RevealVotesRequestObject,
) (generated.RevealVotesResponseObject, error) {
	actor, err := middleware.Participant(ctx)
	if err != nil {
		return nil, err
	}

	view, err := controller.service.Reveal(ctx, actor)
	if err != nil {
		return nil, err
	}

	return generated.RevealVotes200JSONResponse(presenter.RoomState(view)), nil
}

func valueOf(pointer *string) string {
	if pointer == nil {
		return ""
	}

	return *pointer
}
