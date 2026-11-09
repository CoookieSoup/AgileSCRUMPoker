package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/config"
)

func lookupFrom(values map[string]string) config.Lookup {
	return func(key string) (string, bool) {
		found, exists := values[key]

		return found, exists
	}
}

func TestLoadDefaults(t *testing.T) {
	settings, err := config.Load(lookupFrom(map[string]string{"DATABASE_DSN": "dsn"}))

	require.NoError(t, err)
	assert.Equal(t, config.Config{
		DatabaseDSN:     "dsn",
		HTTPAddress:     ":8080",
		AllowedOrigins:  []string{"http://localhost:5173"},
		CleanupInterval: time.Minute,
	}, settings)
}

func TestLoadOverrides(t *testing.T) {
	settings, err := config.Load(lookupFrom(map[string]string{
		"DATABASE_DSN":     "dsn",
		"HTTP_ADDRESS":     ":9000",
		"ALLOWED_ORIGINS":  " https://poker.example.com , http://localhost:3000 ,",
		"CLEANUP_INTERVAL": "15s",
	}))

	require.NoError(t, err)
	assert.Equal(t, ":9000", settings.HTTPAddress)
	assert.Equal(t, []string{"https://poker.example.com", "http://localhost:3000"}, settings.AllowedOrigins)
	assert.Equal(t, []string{"poker.example.com", "localhost:3000"}, settings.OriginPatterns())
	assert.Equal(t, 15*time.Second, settings.CleanupInterval)
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	_, err := config.Load(lookupFrom(map[string]string{}))

	require.ErrorIs(t, err, config.ErrMissingDatabase)

	_, err = config.Load(lookupFrom(map[string]string{"DATABASE_DSN": "dsn", "CLEANUP_INTERVAL": "-1s"}))

	require.Error(t, err)
}

func TestWildcardOriginPattern(t *testing.T) {
	settings := config.Config{AllowedOrigins: []string{"*"}}

	assert.Equal(t, []string{"*"}, settings.OriginPatterns())
}
