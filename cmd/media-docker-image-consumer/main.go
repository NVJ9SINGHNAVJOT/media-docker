// Command media-docker-image-consumer compresses uploaded images to JPEG.
package main

import (
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp"
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp/handlers"
)

func main() {
	consumerapp.Run(consumerapp.Config{
		Service:        "media-docker-image-consumer",
		EnvFile:        ".env.image",
		Topic:          "image",
		RequiresFFmpeg: true,
		Handler:        handlers.Image,
	})
}
