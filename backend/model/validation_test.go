package model_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

func TestNormalizeDisplayName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{name: "trims spaces", input: "  Ona  ", expected: "Ona"},
		{name: "accepts lithuanian letters", input: "Žydrūnė", expected: "Žydrūnė"},
		{name: "accepts 32 runes", input: strings.Repeat("ą", 32), expected: strings.Repeat("ą", 32)},
		{name: "rejects empty", input: "   ", wantErr: true},
		{name: "rejects 33 runes", input: strings.Repeat("a", 33), wantErr: true},
		{name: "rejects control characters", input: "Ona\nJonas", wantErr: true},
		{name: "rejects invalid utf8", input: "\xff\xfe", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := model.NormalizeDisplayName(testCase.input)
			if testCase.wantErr {
				require.ErrorIs(t, err, model.ErrInvalidDisplayName)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestNormalizeCode(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{name: "uppercases and trims", input: " abc234 ", expected: "ABC234"},
		{name: "rejects short code", input: "ABC23", wantErr: true},
		{name: "rejects ambiguous symbols", input: "ABC0O1", wantErr: true},
		{name: "rejects punctuation", input: "ABC-23", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := model.NormalizeCode(testCase.input)
			if testCase.wantErr {
				require.ErrorIs(t, err, model.ErrRoomNotFound)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestParseCard(t *testing.T) {
	for _, card := range model.Deck {
		parsed, err := model.ParseCard(string(card))

		require.NoError(t, err)
		assert.Equal(t, card, parsed)
	}

	for _, input := range []string{"", "4", "0", "100", "?", "coffee"} {
		_, err := model.ParseCard(input)

		require.ErrorIs(t, err, model.ErrInvalidCard, input)
	}
}

func TestNormalizeJiraKey(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{name: "empty means none", input: "  ", expected: ""},
		{name: "accepts key", input: "ABC-123", expected: "ABC-123"},
		{name: "uppercases key", input: "abc-7", expected: "ABC-7"},
		{name: "accepts digits in project", input: "AB2_X-9", expected: "AB2_X-9"},
		{name: "rejects missing number", input: "ABC-", wantErr: true},
		{name: "rejects leading zero", input: "ABC-012", wantErr: true},
		{name: "rejects single letter project", input: "A-1", wantErr: true},
		{name: "rejects url", input: "https://jira.example.com/browse/ABC-1", wantErr: true},
		{name: "rejects too long", input: strings.Repeat("A", 30) + "-123", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := model.NormalizeJiraKey(testCase.input)
			if testCase.wantErr {
				require.ErrorIs(t, err, model.ErrInvalidJiraKey)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestNormalizeLabel(t *testing.T) {
	label, err := model.NormalizeLabel("  Login page  ")

	require.NoError(t, err)
	assert.Equal(t, "Login page", label)

	_, err = model.NormalizeLabel(strings.Repeat("a", 201))

	require.ErrorIs(t, err, model.ErrInvalidLabel)
}

func TestHashTokenIsStableHex(t *testing.T) {
	first := model.HashToken("secret")

	assert.Len(t, first, 64)
	assert.Equal(t, first, model.HashToken("secret"))
	assert.NotEqual(t, first, model.HashToken("other"))
}
