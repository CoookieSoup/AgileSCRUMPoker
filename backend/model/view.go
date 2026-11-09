package model

import "time"

type ParticipantView struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	IsHost      bool   `json:"isHost"`
	HasVoted    bool   `json:"hasVoted"`
}

type VoteView struct {
	ParticipantID string `json:"participantId"`
	Card          Card   `json:"card"`
}

type RoundView struct {
	Label    string     `json:"label"`
	JiraKey  string     `json:"jiraKey,omitempty"`
	Revealed bool       `json:"revealed"`
	Votes    []VoteView `json:"votes"`
}

type RoomView struct {
	Code         string
	ExpiresAt    time.Time
	Participants []ParticipantView
	Round        *RoundView
	OwnVote      Card
}

type Session struct {
	ParticipantID string
	Token         string
	Room          RoomView
}

func NewRoomView(room Room, viewerID string) RoomView {
	cards := map[string]Card{}

	if room.Round != nil {
		for _, vote := range room.Round.Votes {
			cards[vote.ParticipantID] = vote.Card
		}
	}

	view := RoomView{
		Code:         room.Code,
		ExpiresAt:    room.ExpiresAt,
		Participants: make([]ParticipantView, 0, len(room.Participants)),
	}

	for _, participant := range room.Participants {
		_, hasVoted := cards[participant.ID]
		view.Participants = append(view.Participants, NewParticipantView(participant, hasVoted))
	}

	if room.Round == nil {
		return view
	}

	roundView := RoundView{
		Label:    room.Round.Label,
		JiraKey:  room.Round.JiraKey,
		Revealed: room.Round.Revealed,
		Votes:    []VoteView{},
	}

	if room.Round.Revealed {
		roundView.Votes = RevealedVotes(room)
	}

	view.Round = &roundView
	view.OwnVote = cards[viewerID]

	return view
}

func NewParticipantView(participant Participant, hasVoted bool) ParticipantView {
	return ParticipantView{
		ID:          participant.ID,
		DisplayName: participant.DisplayName,
		IsHost:      participant.IsHost,
		HasVoted:    hasVoted,
	}
}

func RevealedVotes(room Room) []VoteView {
	votes := []VoteView{}
	if room.Round == nil {
		return votes
	}

	cards := map[string]Card{}
	for _, vote := range room.Round.Votes {
		cards[vote.ParticipantID] = vote.Card
	}

	for _, participant := range room.Participants {
		card, hasVoted := cards[participant.ID]
		if hasVoted {
			votes = append(votes, VoteView{ParticipantID: participant.ID, Card: card})
		}
	}

	return votes
}
