// Command media-docker-audio-consumer converts uploaded audio files to MP3.
package main

import (
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp"
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp/handlers"
)

func main() {
	consumerapp.Run(consumerapp.Config{
		Service:        "media-docker-audio-consumer",
		EnvFile:        ".env.audio",
		Topic:          "audio",
		RequiresFFmpeg: true,
		Handler:        handlers.Audio,
	})
}
