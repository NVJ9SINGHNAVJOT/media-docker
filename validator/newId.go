package validator

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// newIdMessage represents the structure for extracting the 'NewId' field from JSON data.
type newIdMessage struct {
	NewId string `json:"newId"` // The unique identifier in the JSON message
}

// newIdAndOriginalTopicsMessage represents the structure for extracting both 'NewId' and 'OriginalTopic' fields from JSON data.
type newIdAndOriginalTopicsMessage struct {
	NewId         string `json:"newId"`         // The unique identifier in the JSON message
	OriginalTopic string `json:"originalTopic"` // The topic from which the message originated
}

// ValidateAndParseUUID checks if the provided string is a valid UUID of version 4.
// It returns an error if the UUID is invalid or not version 4.
func ValidateAndParseUUID(idStr string) error {
	// Return an error if NewId is empty
	if idStr == "" {
		return fmt.Errorf("NewId not found")
	}

	// Attempt to parse the UUID string
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err) // Error if parsing fails
	}

	// Ensure the UUID is version 4
	if id.Version() != 4 {
		return fmt.Errorf("NewId is not a UUID v4") // Error if not version 4
	}

	return nil // UUID is valid
}

// ExtractNewId retrieves the 'NewId' field from a JSON byte slice.
// It validates that the NewId is a UUID v4 and returns an error if validation fails.
func ExtractNewId(value []byte) (string, error) {
	var newIdMsg newIdMessage // Struct to hold the NewId

	// Unmarshal the JSON data into the struct
	err := json.Unmarshal(value, &newIdMsg)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err) // Error if JSON parsing fails
	}

	// Validate the NewId field
	err = ValidateAndParseUUID(newIdMsg.NewId)
	if err != nil {
		return "", err // Return validation error
	}

	return newIdMsg.NewId, nil // Return the valid NewId
}

// ExtractNewIdAndOriginalTopic retrieves 'NewId' and 'OriginalTopic' fields from a JSON byte slice.
// It validates that the NewId is a UUID v4 and returns both values, or an error if validation fails.
func ExtractNewIdAndOriginalTopic(value []byte) (string, string, error) {
	var newIdMsg newIdAndOriginalTopicsMessage // Struct to hold NewId and OriginalTopic

	// Unmarshal the JSON data into the struct
	err := json.Unmarshal(value, &newIdMsg)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse JSON: %w", err) // Error if JSON parsing fails
	}

	// Validate the NewId field
	err = ValidateAndParseUUID(newIdMsg.NewId)
	if err != nil {
		return "", "", err // Return validation error
	}

	// Ensure OriginalTopic is not empty
	if newIdMsg.OriginalTopic == "" {
		return "", "", fmt.Errorf("original topic not found") // Error if OriginalTopic is missing
	}

	return newIdMsg.NewId, newIdMsg.OriginalTopic, nil // Return both the NewId and OriginalTopic
}
