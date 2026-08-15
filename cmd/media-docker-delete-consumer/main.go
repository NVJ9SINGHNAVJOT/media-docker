// Command media-docker-delete-consumer removes assets from media storage.
//
// Deletion is type-agnostic because every asset is a single directory, so this
// one service owns the delete-file topic for all media types. It performs no
// conversion and therefore needs no ffmpeg in its runtime image.
package main

import (
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp"
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp/handlers"
)

func main() {
	consumerapp.Run(consumerapp.Config{
		Service:    "media-docker-delete-consumer",
		EnvFile:    ".env.delete",
		Topic:      "delete-file",
		RawHandler: handlers.DeleteFile,
	})
}
