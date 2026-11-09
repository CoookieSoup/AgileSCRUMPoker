package round

import (
	"context"
	"fmt"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

type Service struct {
	repository Repository
	publisher  Publisher
	generator  IDGenerator
	clock      Clock
}

func New(repository Repository, publisher Publisher, generator IDGenerator, clock Clock) Service {
	return Service{
		repository: repository,
		publisher:  publisher,
		generator:  generator,
		clock:      clock,
	}
}

func (service Service) Start(
	ctx context.Context,
	actor model.Participant,
	label string,
	jiraKey string,
) (model.RoomView, error) {
	if !actor.IsHost {
		return model.RoomView{}, model.ErrNotHost
	}

	round, err := service.newRound(actor.RoomCode, label, jiraKey)
	if err != nil {
		return model.RoomView{}, err
	}

	err = service.repository.Replace(ctx, round)
	if err != nil {
		return model.RoomView{}, fmt.Errorf("replace round: %w", err)
	}

	service.publisher.Publish(actor.RoomCode, model.RoundStartedEvent(round))

	return service.view(ctx, actor)
}

func (service Service) Vote(ctx context.Context, actor model.Participant, card string) error {
	parsedCard, err := model.ParseCard(card)
	if err != nil {
		return err
	}

	err = service.repository.UpsertVote(ctx, actor.RoomCode, actor.ID, parsedCard)
	if err != nil {
		return fmt.Errorf("store vote: %w", err)
	}

	service.publisher.Publish(actor.RoomCode, model.ParticipantVotedEvent(actor.ID))

	return nil
}

func (service Service) Reveal(ctx context.Context, actor model.Participant) (model.RoomView, error) {
	if !actor.IsHost {
		return model.RoomView{}, model.ErrNotHost
	}

	err := service.repository.Reveal(ctx, actor.RoomCode)
	if err != nil {
		return model.RoomView{}, fmt.Errorf("reveal round: %w", err)
	}

	room, err := service.repository.Load(ctx, actor.RoomCode, service.clock.Now())
	if err != nil {
		return model.RoomView{}, fmt.Errorf("load room: %w", err)
	}

	service.publisher.Publish(actor.RoomCode, model.VotesRevealedEvent(model.RevealedVotes(room)))

	return model.NewRoomView(room, actor.ID), nil
}

func (service Service) newRound(code string, label string, jiraKey string) (model.Round, error) {
	normalizedLabel, err := model.NormalizeLabel(label)
	if err != nil {
		return model.Round{}, err
	}

	normalizedKey, err := model.NormalizeJiraKey(jiraKey)
	if err != nil {
		return model.Round{}, err
	}

	roundID, err := service.generator.NewID()
	if err != nil {
		return model.Round{}, fmt.Errorf("generate round id: %w", err)
	}

	round := model.Round{
		ID:       roundID,
		RoomCode: code,
		Label:    normalizedLabel,
		JiraKey:  normalizedKey,
	}

	return round, nil
}

func (service Service) view(ctx context.Context, actor model.Participant) (model.RoomView, error) {
	room, err := service.repository.Load(ctx, actor.RoomCode, service.clock.Now())
	if err != nil {
		return model.RoomView{}, fmt.Errorf("load room: %w", err)
	}

	return model.NewRoomView(room, actor.ID), nil
}
