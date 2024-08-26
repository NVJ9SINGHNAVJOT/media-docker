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
	Message string `json:"message" validate:"required"`
	Data    any    `json:"data,omitempty"`
}

func Response(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	if status > 299 {
		if data != nil {
			log.Error().Any("error", data).Msg(message)
		} else {
			log.Error().Msg(message)
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
	}
	if data != nil {
		response.Data = data
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error encoding to json")
	}
}
