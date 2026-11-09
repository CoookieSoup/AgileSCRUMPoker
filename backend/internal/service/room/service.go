package room

import (
	"context"
	"errors"
	"fmt"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

const maxCodeAttempts = 5

type Service struct {
	repository Repository
	publisher  Publisher
	generator  Generator
	clock      Clock
}

func New(repository Repository, publisher Publisher, generator Generator, clock Clock) Service {
	return Service{
		repository: repository,
		publisher:  publisher,
		generator:  generator,
		clock:      clock,
	}
}

func (service Service) Create(ctx context.Context, displayName string) (model.Session, error) {
	host, token, err := service.newParticipant(displayName, true)
	if err != nil {
		return model.Session{}, err
	}

	room, err := service.createRoom(ctx, host)
	if err != nil {
		return model.Session{}, err
	}

	return newSession(room, room.Participants[0], token), nil
}

func (service Service) Join(ctx context.Context, code string, displayName string) (model.Session, error) {
	roomCode, err := model.NormalizeCode(code)
	if err != nil {
		return model.Session{}, err
	}

	participant, token, err := service.newParticipant(displayName, false)
	if err != nil {
		return model.Session{}, err
	}

	participant.RoomCode = roomCode

	err = service.repository.AddParticipant(ctx, participant, participant.CreatedAt)
	if err != nil {
		return model.Session{}, fmt.Errorf("add participant: %w", err)
	}

	service.publisher.Publish(roomCode, model.ParticipantJoinedEvent(participant))

	room, err := service.repository.Load(ctx, roomCode, participant.CreatedAt)
	if err != nil {
		return model.Session{}, fmt.Errorf("load room: %w", err)
	}

	return newSession(room, participant, token), nil
}

func (service Service) Get(ctx context.Context, actor model.Participant) (model.RoomView, error) {
	room, err := service.repository.Load(ctx, actor.RoomCode, service.clock.Now())
	if err != nil {
		return model.RoomView{}, fmt.Errorf("load room: %w", err)
	}

	return model.NewRoomView(room, actor.ID), nil
}

func (service Service) Leave(ctx context.Context, actor model.Participant) error {
	departure, err := service.repository.RemoveParticipant(ctx, actor.RoomCode, actor.ID)
	if err != nil {
		return fmt.Errorf("remove participant: %w", err)
	}

	if departure.RoomDeleted {
		service.publisher.CloseRoom(actor.RoomCode, model.CloseReasonClosed)

		return nil
	}

	service.publisher.Publish(actor.RoomCode, model.ParticipantLeftEvent(actor.ID))

	if departure.NewHostID != "" {
		service.publisher.Publish(actor.RoomCode, model.HostChangedEvent(departure.NewHostID))
	}

	return nil
}

func (service Service) Close(ctx context.Context, actor model.Participant) error {
	if !actor.IsHost {
		return model.ErrNotHost
	}

	err := service.repository.Delete(ctx, actor.RoomCode)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}

	service.publisher.CloseRoom(actor.RoomCode, model.CloseReasonClosed)

	return nil
}

func (service Service) createRoom(ctx context.Context, host model.Participant) (model.Room, error) {
	for range maxCodeAttempts {
		code, err := service.generator.NewCode()
		if err != nil {
			return model.Room{}, fmt.Errorf("generate room code: %w", err)
		}

		host.RoomCode = code
		room := model.Room{
			Code:         code,
			ExpiresAt:    host.CreatedAt.Add(model.RoomLifetime),
			CreatedAt:    host.CreatedAt,
			Participants: []model.Participant{host},
		}

		err = service.repository.Create(ctx, room)
		if errors.Is(err, model.ErrCodeTaken) {
			continue
		}

		if err != nil {
			return model.Room{}, fmt.Errorf("create room: %w", err)
		}

		return room, nil
	}

	return model.Room{}, model.ErrCodeTaken
}

func (service Service) newParticipant(displayName string, isHost bool) (model.Participant, string, error) {
	name, err := model.NormalizeDisplayName(displayName)
	if err != nil {
		return model.Participant{}, "", err
	}

	participantID, err := service.generator.NewID()
	if err != nil {
		return model.Participant{}, "", fmt.Errorf("generate participant id: %w", err)
	}

	token, err := service.generator.NewToken()
	if err != nil {
		return model.Participant{}, "", fmt.Errorf("generate session token: %w", err)
	}

	participant := model.Participant{
		ID:          participantID,
		DisplayName: name,
		TokenHash:   model.HashToken(token),
		IsHost:      isHost,
		CreatedAt:   service.clock.Now(),
	}

	return participant, token, nil
}

func newSession(room model.Room, participant model.Participant, token string) model.Session {
	return model.Session{
		ParticipantID: participant.ID,
		Token:         token,
		Room:          model.NewRoomView(room, participant.ID),
	}
}
