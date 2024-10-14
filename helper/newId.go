package helper

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// newIdMessage represents the structure of the message with only the NewId field.
type newIdMessage struct {
	NewId string `json:"newId"` // Field to extract NewId
}

// ExtractNewId extracts the NewId field from a JSON-encoded byte slice.
// It returns the NewId if valid (UUID v4), or an error if not.
func ExtractNewId(value []byte) (string, error) {
	var newIdMsg newIdMessage
	if err := json.Unmarshal(value, &newIdMsg); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if newIdMsg.NewId == "" {
		return "", fmt.Errorf("NewId not found")
	}

	id, err := uuid.Parse(newIdMsg.NewId)
	if err != nil {
		return "", fmt.Errorf("invalid UUID format: %w", err)
	}

	if id.Version() != 4 {
		return "", fmt.Errorf("NewId is not a UUID v4")
	}

	return newIdMsg.NewId, nil // Return the validated NewId
}
