package helper

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// apiResponse defines the structure of the JSON response sent to the client.
type apiResponse struct {
	Message string `json:"message" validate:"required"` // Human-readable message about the response status.
	Data    any    `json:"data,omitempty"`              // Optional field for additional data, if any.
}

// Response sends a JSON response to the client with the specified status code and message.
// It handles error logging for various scenarios, ensuring proper feedback is given to the client
// and that internal errors are logged for debugging.
func Response(w http.ResponseWriter, status int, message string, data any) {
	// Prepare the response structure with the provided message.
	response := apiResponse{
		Message: message,
	}

	// Log error details if the status indicates a client or server error (status code > 299).
	if status > 299 {
		logError(status, message, data) // Log error information
	} else if data != nil {
		// If status is OK and data is not nil, include it in the response.
		response.Data = data
	}

	// Attempt to encode the response to JSON.
	encodedResponse, err := json.Marshal(response)
	if err != nil {
		// Log the encoding error and respond with a 500 Internal Server Error.
		log.Error().Err(err).Any("Response", response).Msg("Error encoding response to JSON")
		http.Error(w, `{"message":"internal server error, while encoding json"}`, http.StatusInternalServerError)
		return // Exit to prevent further processing.
	}

	// Set headers and write the status code for the response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Write the encoded response to the ResponseWriter.
	if _, err := w.Write(encodedResponse); err != nil {
		// Log any error that occurs while writing the response.
		log.Error().Err(err).Any("encodedResponse", encodedResponse).Msg("Error writing response")
		// NOTE: Writing to the response is already done; we can't change it now.
	}
}

// logError handles logging of errors with the provided status code and message.
func logError(status int, message string, data any) {
	if data != nil {
		if err, ok := data.(error); ok {
			log.Error().Int("status", status).Err(err).Msg(message) // Log error if it's of type error.
		} else {
			log.Error().Int("status", status).Any("data", data).Msg(message) // Log non-error data if provided.
		}
	} else {
		log.Error().Int("status", status).Msg(message) // Log the message alone if no data is provided.
	}
}
