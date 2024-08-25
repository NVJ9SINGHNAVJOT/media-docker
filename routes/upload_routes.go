package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/middleware"
)

func UploadRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		router.Post("/video", middleware.FileStorage("videoFile", http.HandlerFunc(api.UploadVideo)))
	}
}
