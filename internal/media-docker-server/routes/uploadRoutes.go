package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
)

func UploadRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		router.Post("/chunksStorage", api.ChunksStorage)
		router.Post("/fileStorage", api.FileStorage)
		router.Post("/video", api.Video)
		router.Post("/videoResolutions", api.VideoResolutions)
		router.Post("/image", api.Image)
		router.Post("/audio", api.Audio)
	}
}
