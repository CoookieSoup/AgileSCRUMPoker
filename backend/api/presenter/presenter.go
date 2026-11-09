package presenter

import (
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/api/generated"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

func Session(session model.Session) generated.Session {
	return generated.Session{
		ParticipantId: session.ParticipantID,
		SessionToken:  session.Token,
		Room:          RoomState(session.Room),
	}
}

func RoomState(view model.RoomView) generated.RoomState {
	state := generated.RoomState{
		Code:         view.Code,
		ExpiresAt:    view.ExpiresAt,
		Participants: make([]generated.Participant, 0, len(view.Participants)),
	}

	for _, participant := range view.Participants {
		state.Participants = append(state.Participants, generated.Participant{
			Id:          participant.ID,
			DisplayName: participant.DisplayName,
			IsHost:      participant.IsHost,
			HasVoted:    participant.HasVoted,
		})
	}

	if view.Round != nil {
		round := roundState(*view.Round)
		state.Round = &round
	}

	if view.OwnVote != "" {
		ownVote := string(view.OwnVote)
		state.OwnVote = &ownVote
	}

	return state
}

func roundState(view model.RoundView) generated.Round {
	round := generated.Round{
		Label:    view.Label,
		Revealed: view.Revealed,
		Votes:    make([]generated.Vote, 0, len(view.Votes)),
	}

	if view.JiraKey != "" {
		jiraKey := view.JiraKey
		round.JiraKey = &jiraKey
	}

	for _, vote := range view.Votes {
		round.Votes = append(round.Votes, generated.Vote{ParticipantId: vote.ParticipantID, Card: string(vote.Card)})
	}

	return round
}
