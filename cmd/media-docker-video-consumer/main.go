// Command media-docker-video-consumer converts uploaded videos to a
// single-quality HLS stream.
package main

import (
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp"
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp/handlers"
)

func main() {
	consumerapp.Run(consumerapp.Config{
		Service:        "media-docker-video-consumer",
		EnvFile:        ".env.video",
		Topic:          "video",
		RequiresFFmpeg: true,
		Handler:        handlers.Video,
	})
}
