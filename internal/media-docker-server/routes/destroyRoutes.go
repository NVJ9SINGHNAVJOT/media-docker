package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
)

func DestroyRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		router.Delete("/delete-file", http.HandlerFunc(api.DeleteFile))
	}
}
