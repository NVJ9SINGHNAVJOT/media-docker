package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/api"
)

func UploadRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		// Storage: write the uploaded bytes into media storage.
		router.Post("/chunks-storage", api.ChunksStorage)
		router.Post("/file-storage", api.FileStorage)

		// Dispatch: claim a stored upload and queue its conversion job.
		router.Post("/video", api.Video)
		router.Post("/video-resolutions", api.VideoResolutions)
		router.Post("/image", api.Image)
		router.Post("/audio", api.Audio)

		// Stored as-is; no conversion job is queued for these.
		router.Post("/document", api.Document)
		router.Post("/other", api.Other)
	}
}
