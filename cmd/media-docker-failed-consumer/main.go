// Command media-docker-failed-consumer retries conversions that failed.
//
// It consumes the dead-letter queue every other consumer writes to, and is the
// last attempt made at converting an asset. Since v4 that is no longer a
// make-or-break decision: an asset whose conversion never succeeds keeps serving
// the raw upload it was created with.
package main

import (
	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-failed-consumer/process"
)

func main() {
	consumerapp.Run(consumerapp.Config{
		Service:        "media-docker-failed-consumer",
		EnvFile:        ".env.failed",
		Topic:          "failed-letter-queue",
		RequiresFFmpeg: true,
		// Retries must not be re-queued to the DLQ they came from, so they
		// bypass the standard wrapper.
		RawHandler: process.ProcessMessage,
	})
}
