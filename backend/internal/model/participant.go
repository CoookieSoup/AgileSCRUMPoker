package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxDisplayNameLength = 32

type Participant struct {
	ID          string    `gorm:"primaryKey;size:32"`
	RoomCode    string    `gorm:"not null;size:6;index"`
	DisplayName string    `gorm:"not null;size:128"`
	TokenHash   string    `gorm:"not null;size:64;uniqueIndex"`
	IsHost      bool      `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	Votes       []Vote    `gorm:"foreignKey:ParticipantID;references:ID;constraint:OnDelete:CASCADE"`
}

func NormalizeDisplayName(value string) (string, error) {
	name := strings.TrimSpace(value)

	length := utf8.RuneCountInString(name)
	if length == 0 || length > MaxDisplayNameLength || !isPrintable(name) {
		return "", ErrInvalidDisplayName
	}

	return name, nil
}

func HashToken(token string) string {
	digest := sha256.Sum256([]byte(token))

	return hex.EncodeToString(digest[:])
}

func isPrintable(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}

	for _, symbol := range value {
		if !unicode.IsPrint(symbol) {
			return false
		}
	}

	return true
}
