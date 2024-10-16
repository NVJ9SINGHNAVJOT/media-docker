package helper

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// newIdMessage defines the structure for extracting the NewId field from JSON.
type newIdMessage struct {
	NewId string `json:"newId"` // JSON field containing the NewId
}

// newIdAndOriginalTopicsMessage defines the structure for extracting the NewId and OriginalTopic fields from JSON.
type newIdAndOriginalTopicsMessage struct {
	NewId         string `json:"newId"`         // JSON field containing the NewId
	OriginalTopic string `json:"originalTopic"` // JSON field indicating the source topic of the message
}

// validateAndParseUUID verifies if the provided UUID string is valid and of version 4.
// It returns the UUID as a string if valid, or an error if invalid.
func validateAndParseUUID(idStr string) (string, error) {
	// Return an error if the NewId is empty
	if idStr == "" {
		return "", fmt.Errorf("NewId not found")
	}

	// Attempt to parse the UUID string
	id, err := uuid.Parse(idStr)
	if err != nil {
		return "", fmt.Errorf("invalid UUID format: %w", err) // Return an error if parsing fails
	}

	// Check if the UUID version is 4
	if id.Version() != 4 {
		return "", fmt.Errorf("NewId is not a UUID v4") // Return an error if the UUID is not version 4
	}

	return idStr, nil // Return the validated UUID string
}

// ExtractNewId extracts the NewId field from a JSON-encoded byte slice.
// It returns the NewId if valid (UUID v4) or an error if not.
func ExtractNewId(value []byte) (string, error) {
	var newIdMsg newIdMessage // Create an instance of newIdMessage

	// Unmarshal the JSON data into newIdMsg
	if err := json.Unmarshal(value, &newIdMsg); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err) // Return an error if unmarshalling fails
	}

	// Validate and parse the UUID before returning
	return validateAndParseUUID(newIdMsg.NewId) // Return the validated NewId
}

// ExtractNewIdAndOriginalTopic retrieves the NewId and OriginalTopic fields from a JSON-encoded byte slice.
// It returns the NewId if valid (UUID v4) along with the OriginalTopic, or an error if validation fails.
func ExtractNewIdAndOriginalTopic(value []byte) (newId string, originalTopic string, err error) {
	var newIdMsg newIdAndOriginalTopicsMessage // Create an instance of newIdAndOriginalTopicsMessage

	// Unmarshal the JSON data into newIdMsg
	if err := json.Unmarshal(value, &newIdMsg); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal JSON: %w", err) // Return an error if unmarshalling fails
	}

	// Validate the NewId field
	_, err = validateAndParseUUID(newIdMsg.NewId) // Validate the NewId
	if err != nil {
		return "", "", err // Return the error if validation fails
	}

	// Return an error if OriginalTopic is empty
	if newIdMsg.OriginalTopic == "" {
		return "", "", fmt.Errorf("original topic not found")
	}

	// Return the NewId and OriginalTopic
	return newIdMsg.NewId, newIdMsg.OriginalTopic, nil // Return both values
}
