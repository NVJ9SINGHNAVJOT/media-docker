package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
)

func StatusRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		router.Post("/status", http.HandlerFunc(api.Status))
	}
}
