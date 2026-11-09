package model

import (
	"strings"
	"time"
)

const (
	MaxParticipants = 20
	RoomLifetime    = time.Hour
	CodeLength      = 6
	CodeAlphabet    = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

type Room struct {
	Code         string        `gorm:"primaryKey;size:6"`
	ExpiresAt    time.Time     `gorm:"not null;index"`
	CreatedAt    time.Time     `gorm:"not null"`
	Participants []Participant `gorm:"foreignKey:RoomCode;references:Code;constraint:OnDelete:CASCADE"`
	Round        *Round        `gorm:"foreignKey:RoomCode;references:Code;constraint:OnDelete:CASCADE"`
}

func (room Room) IsExpired(now time.Time) bool {
	return !now.Before(room.ExpiresAt)
}

func (room Room) Participant(participantID string) (Participant, bool) {
	for _, participant := range room.Participants {
		if participant.ID == participantID {
			return participant, true
		}
	}

	return Participant{}, false
}

func NormalizeCode(value string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) != CodeLength {
		return "", ErrRoomNotFound
	}

	for _, symbol := range code {
		if !strings.ContainsRune(CodeAlphabet, symbol) {
			return "", ErrRoomNotFound
		}
	}

	return code, nil
}
