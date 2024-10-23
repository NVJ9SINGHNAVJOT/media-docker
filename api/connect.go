package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/rs/zerolog/log"
)

// Connect handles incoming HTTP requests to establish a connection
// Logs an informational message and sends a response with HTTP status 200
func Connect(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("connection established")                         // Log the successful connection establishment
	helper.Response(w, http.StatusOK, "connection established", nil) // Send a successful response back to the client
}
