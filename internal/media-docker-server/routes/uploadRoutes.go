package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/middleware"
)

func UploadRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		router.Post("/video", middleware.FileStorage("videoFile", http.HandlerFunc(api.Video)))
		router.Post("/videoResolutions", middleware.FileStorage("videoFile", http.HandlerFunc(api.VideoResolutions)))
		router.Post("/image", middleware.FileStorage("imageFile", http.HandlerFunc(api.Image)))
		router.Post("/audio", middleware.FileStorage("audioFile", http.HandlerFunc(api.Audio)))
	}
}
