package random

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/model"
)

const (
	idBytes    = 16
	tokenBytes = 32
)

type Generator struct{}

func (Generator) NewCode() (string, error) {
	buffer := make([]byte, model.CodeLength)

	_, err := rand.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	alphabetSize := byte(len(model.CodeAlphabet))
	for index, value := range buffer {
		buffer[index] = model.CodeAlphabet[value%alphabetSize]
	}

	return string(buffer), nil
}

func (Generator) NewID() (string, error) {
	buffer := make([]byte, idBytes)

	_, err := rand.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return hex.EncodeToString(buffer), nil
}

func (Generator) NewToken() (string, error) {
	buffer := make([]byte, tokenBytes)

	_, err := rand.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
