package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
)

func ConnectionRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/connect", http.HandlerFunc(api.Connect))
	}
}
