// Command media-docker-video-resolutions-consumer converts uploaded videos into
// the full resolution ladder plus an adaptive master playlist.
//
// This is the most expensive job in the system -- one transcode per rung -- which
// is precisely why it runs as its own service and scales independently.
package main

import (
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp"
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp/handlers"
)

func main() {
	consumerapp.Run(consumerapp.Config{
		Service:        "media-docker-video-resolutions-consumer",
		EnvFile:        ".env.video-resolutions",
		Topic:          "video-resolutions",
		RequiresFFmpeg: true,
		Handler:        handlers.VideoResolutions,
	})
}
