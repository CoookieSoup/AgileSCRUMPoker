package random_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/infra/random"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/internal/model"
)

func TestNewCodeIsValidRoomCode(t *testing.T) {
	generator := random.Generator{}

	for range 200 {
		code, err := generator.NewCode()

		require.NoError(t, err)

		normalized, err := model.NormalizeCode(code)

		require.NoError(t, err)
		assert.Equal(t, code, normalized)
	}
}

func TestNewIDAndTokenAreUniqueAndSized(t *testing.T) {
	generator := random.Generator{}
	seen := map[string]bool{}

	for range 100 {
		identifier, err := generator.NewID()

		require.NoError(t, err)
		assert.Len(t, identifier, 32)

		token, err := generator.NewToken()

		require.NoError(t, err)
		assert.Len(t, token, 43)
		assert.False(t, seen[identifier])
		assert.False(t, seen[token])

		seen[identifier] = true
		seen[token] = true
	}
}
