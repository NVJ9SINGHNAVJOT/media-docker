package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

// dirDeleteChan queues directories for background removal, buffered so that a
// request handler never blocks on a delete.
//
// There is no matching channel for single files. Nothing deletes an individual
// file out of an asset: a conversion only ever adds to an asset directory, and
// removing an asset removes the directory whole.
var dirDeleteChan = make(chan string, 1000)

// DeleteDirWorker listens on the dirDeleteChan and deletes directories as requested
//
// NOTE: It is important to call this function within a goroutine.
func DeleteDirWorker() {
	for path := range dirDeleteChan {
		err := os.RemoveAll(path)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("error deleting directory")
		}
	}
}

// AddToDirDeleteChan sends a directory path to the dirDeleteChan with a non-blocking operation.
// If the channel is full, it logs a warning.
func AddToDirDeleteChan(path string) {
	select {
	case dirDeleteChan <- path:
		// Successfully added to the channel
	default:
		// Channel is full, log a warning
		log.Warn().Str("dirPath", path).Msg("directory deletion channel is full, unable to queue directory for deletion")
	}
}

// CloseDeleteChannels closes the directory deletion channel, draining its worker.
func CloseDeleteChannels() {
	close(dirDeleteChan)
}
