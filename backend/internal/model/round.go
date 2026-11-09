package model

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	MaxLabelLength   = 200
	MaxJiraKeyLength = 32
)

var jiraKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]+-[1-9][0-9]*$`)

type Card string

const (
	CardOne       Card = "1"
	CardTwo       Card = "2"
	CardThree     Card = "3"
	CardFive      Card = "5"
	CardEight     Card = "8"
	CardThirteen  Card = "13"
	CardTwentyOne Card = "21"
)

var Deck = []Card{CardOne, CardTwo, CardThree, CardFive, CardEight, CardThirteen, CardTwentyOne}

type Round struct {
	ID       string `gorm:"primaryKey;size:32"`
	RoomCode string `gorm:"not null;size:6;uniqueIndex"`
	Label    string `gorm:"not null;size:800"`
	JiraKey  string `gorm:"not null;size:32"`
	Revealed bool   `gorm:"not null"`
	Votes    []Vote `gorm:"foreignKey:RoundID;references:ID;constraint:OnDelete:CASCADE"`
}

type Vote struct {
	RoundID       string `gorm:"primaryKey;size:32"`
	ParticipantID string `gorm:"primaryKey;size:32"`
	Card          Card   `gorm:"not null;size:2"`
}

func ParseCard(value string) (Card, error) {
	card := Card(strings.TrimSpace(value))
	if !slices.Contains(Deck, card) {
		return "", ErrInvalidCard
	}

	return card, nil
}

func NormalizeLabel(value string) (string, error) {
	label := strings.TrimSpace(value)
	if utf8.RuneCountInString(label) > MaxLabelLength || !isPrintable(label) {
		return "", ErrInvalidLabel
	}

	return label, nil
}

func NormalizeJiraKey(value string) (string, error) {
	jiraKey := strings.ToUpper(strings.TrimSpace(value))
	if jiraKey == "" {
		return "", nil
	}

	if len(jiraKey) > MaxJiraKeyLength || !jiraKeyPattern.MatchString(jiraKey) {
		return "", ErrInvalidJiraKey
	}

	return jiraKey, nil
}
