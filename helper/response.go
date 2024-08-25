package helper

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

type APIResponse struct {
	Message string
}

type APIResponseWithData struct {
	Message string
	Data    any
}

func Response(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	if status > 299 {
		if data != nil {
			log.Error().Any("error", data).Msg(message)
		}
		response := APIResponse{
			Message: message,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error().Str("error", err.Error()).Msg("error encoding to json")
		}
		return
	}

	response := APIResponseWithData{
		Message: message,
		Data:    data,
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error encoding to json")
	}
}
