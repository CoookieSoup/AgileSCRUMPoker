package model

import "errors"

var (
	ErrRoomNotFound       = errors.New("room not found")
	ErrRoomFull           = errors.New("room is full")
	ErrCodeTaken          = errors.New("room code is already taken")
	ErrUnauthorized       = errors.New("session is missing or invalid")
	ErrNotHost            = errors.New("only the host can do this")
	ErrRoundNotFound      = errors.New("no round is in progress")
	ErrRoundRevealed      = errors.New("votes are already revealed")
	ErrInvalidDisplayName = errors.New("display name must be 1 to 32 printable characters")
	ErrInvalidCard        = errors.New("card is not in the deck")
	ErrInvalidLabel       = errors.New("label must be at most 200 printable characters")
	ErrInvalidJiraKey     = errors.New("jira key must look like ABC-123")
	ErrUnavailable        = errors.New("service is unavailable")
)
