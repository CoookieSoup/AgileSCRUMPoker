package model

type EventType string

const (
	EventParticipantJoined EventType = "participant_joined"
	EventParticipantLeft   EventType = "participant_left"
	EventHostChanged       EventType = "host_changed"
	EventRoundStarted      EventType = "round_started"
	EventParticipantVoted  EventType = "participant_voted"
	EventVotesRevealed     EventType = "votes_revealed"
	EventRoomClosed        EventType = "room_closed"
)

type CloseReason string

const (
	CloseReasonClosed  CloseReason = "closed"
	CloseReasonExpired CloseReason = "expired"
)

type Event struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

type ParticipantPayload struct {
	ParticipantID string `json:"participantId"`
}

type RoundStartedPayload struct {
	Label   string `json:"label"`
	JiraKey string `json:"jiraKey,omitempty"`
}

type VotesRevealedPayload struct {
	Votes []VoteView `json:"votes"`
}

type RoomClosedPayload struct {
	Reason CloseReason `json:"reason"`
}

func ParticipantJoinedEvent(participant Participant) Event {
	return Event{Type: EventParticipantJoined, Payload: NewParticipantView(participant, false)}
}

func ParticipantLeftEvent(participantID string) Event {
	return Event{Type: EventParticipantLeft, Payload: ParticipantPayload{ParticipantID: participantID}}
}

func HostChangedEvent(participantID string) Event {
	return Event{Type: EventHostChanged, Payload: ParticipantPayload{ParticipantID: participantID}}
}

func RoundStartedEvent(round Round) Event {
	return Event{Type: EventRoundStarted, Payload: RoundStartedPayload{Label: round.Label, JiraKey: round.JiraKey}}
}

func ParticipantVotedEvent(participantID string) Event {
	return Event{Type: EventParticipantVoted, Payload: ParticipantPayload{ParticipantID: participantID}}
}

func VotesRevealedEvent(votes []VoteView) Event {
	return Event{Type: EventVotesRevealed, Payload: VotesRevealedPayload{Votes: votes}}
}

func RoomClosedEvent(reason CloseReason) Event {
	return Event{Type: EventRoomClosed, Payload: RoomClosedPayload{Reason: reason}}
}
