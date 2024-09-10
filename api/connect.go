package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/rs/zerolog/log"
)

func Connect(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("connection established")
	helper.Response(w, http.StatusOK, "connection established", nil)
}
