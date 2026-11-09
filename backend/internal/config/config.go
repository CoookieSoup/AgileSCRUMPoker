package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var ErrMissingDatabase = errors.New("DATABASE_DSN must be set")

type Config struct {
	DatabaseDSN     string
	HTTPAddress     string
	AllowedOrigins  []string
	CleanupInterval time.Duration
}

type Lookup func(key string) (string, bool)

func Load(lookup Lookup) (Config, error) {
	settings := Config{
		DatabaseDSN:     value(lookup, "DATABASE_DSN", ""),
		HTTPAddress:     value(lookup, "HTTP_ADDRESS", ":8080"),
		AllowedOrigins:  splitList(value(lookup, "ALLOWED_ORIGINS", "http://localhost:5173")),
		CleanupInterval: time.Minute,
	}

	if settings.DatabaseDSN == "" {
		return Config{}, ErrMissingDatabase
	}

	interval, found := lookup("CLEANUP_INTERVAL")
	if found && interval != "" {
		parsed, err := time.ParseDuration(interval)
		if err != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("CLEANUP_INTERVAL must be a positive duration: %q", interval)
		}

		settings.CleanupInterval = parsed
	}

	return settings, nil
}

func (settings Config) OriginPatterns() []string {
	patterns := make([]string, 0, len(settings.AllowedOrigins))

	for _, origin := range settings.AllowedOrigins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" {
			patterns = append(patterns, origin)

			continue
		}

		patterns = append(patterns, parsed.Host)
	}

	return patterns
}

func value(lookup Lookup, key string, fallback string) string {
	found, exists := lookup(key)
	if !exists || strings.TrimSpace(found) == "" {
		return fallback
	}

	return strings.TrimSpace(found)
}

func splitList(commaSeparated string) []string {
	items := []string{}

	for item := range strings.SplitSeq(commaSeparated, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}
