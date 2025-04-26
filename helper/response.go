package helper

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// apiResponse defines the structure of the standard JSON response sent to clients.
type apiResponse struct {
	Message string `json:"message" validate:"required"` // Descriptive message about the result (success or error)
	Data    any    `json:"data,omitempty"`              // Optional data payload for success responses
}

// SuccessResponse sends a standardized JSON success response.
// It automatically logs a warning if a non-2xx HTTP status code is used.
//
// Parameters:
//   - w: HTTP ResponseWriter to write the response
//   - requestId: unique identifier for the request (for logging)
//   - status: HTTP status code (expected to be 2xx)
//   - message: human-readable success message
//   - data: optional data payload (can be nil)
func SuccessResponse(w http.ResponseWriter, requestId string, status int, message string, data any) {
	if status >= 300 {
		log.Warn().Str("requestId", requestId).Int("status", status).Msg("SuccessResponse used with non-2xx status")
	}

	writeJSONResponse(w, requestId, status, apiResponse{
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends a standardized JSON error response.
// It logs the error details using structured logging.
//
// Parameters:
//   - w: HTTP ResponseWriter to write the response
//   - requestId: unique identifier for the request (for logging)
//   - status: HTTP status code (expected to be 4xx or 5xx)
//   - message: human-readable error message
//   - err: optional underlying error (can be nil)
func ErrorResponse(w http.ResponseWriter, requestId string, status int, message string, err error) {
	if status < 400 {
		log.Warn().Str("requestId", requestId).Int("status", status).Msg("ErrorResponse used with non-4xx/5xx status")
	}

	if err != nil {
		log.Error().Str("requestId", requestId).Int("status", status).Err(err).Msg(message)
	} else {
		log.Error().Str("requestId", requestId).Int("status", status).Msg(message)
	}

	writeJSONResponse(w, requestId, status, apiResponse{
		Message: message,
	})
}

// writeJSONResponse serializes the apiResponse into JSON and writes it to the client.
// It ensures that the HTTP status code is only set after successful JSON encoding
// to avoid accidentally sending a wrong status on marshaling failure.
//
// Parameters:
//   - w: HTTP ResponseWriter to write the response
//   - requestId: unique identifier for the request (for logging)
//   - status: HTTP status code to set in the response
//   - resp: the apiResponse object to serialize and send
func writeJSONResponse(w http.ResponseWriter, requestId string, status int, resp apiResponse) {
	w.Header().Set("Content-Type", "application/json")

	encoded, err := json.Marshal(resp)
	if err != nil {
		// If marshaling fails, log the error and send a 500 Internal Server Error.
		log.Error().Err(err).Str("requestId", requestId).Any("response", resp).Msg("Error encoding JSON response")
		http.Error(w, `{"message":"internal server error, while encoding json"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status) // Set status code only after successful encoding

	if _, err := w.Write(encoded); err != nil {
		// Log any failure to write to the client (e.g., client disconnected).
		log.Error().Err(err).Str("requestId", requestId).Msg("Error writing JSON response to client")
	}
}
