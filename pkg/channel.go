package pkg

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Channels for file and directory deletion with a buffer size of 1000
var (
	fileDeleteChan = make(chan string, 1000) // Buffered channel for file paths
	dirDeleteChan  = make(chan string, 1000) // Buffered channel for directory paths
)

const (
	retries  = 3
	waitTime = 1 * time.Second // Constant 1 second delay between retries
)

// DeleteFileWorker listens on the fileDeleteChan and deletes files as requested with retries
//
// This function gives a total of 3 attempts for deleting the file. If it fails after 2 attempts,
// the 3rd attempt will log an error. The retry wait time is fixed at 1 second.
//
// NOTE: It is important to call this function within a goroutine.
func DeleteFileWorker() {
	for path := range fileDeleteChan {

		for i := 1; i <= retries; i++ {
			err := os.Remove(path)
			if err == nil {
				break // Successfully deleted, no need to retry
			}

			// If the error is because the file is in use, retry
			if i < retries && strings.Contains(err.Error(), "The process cannot access the file because it is being used by another process.") {
				log.Warn().Msgf("Path: %s, Attempt %d to delete file failed because it is in use. Retrying in 1 second...", path, i)
				time.Sleep(waitTime) // Constant 1-second wait before retrying
			} else {
				// Log error if it's not a file-in-use error or max retries exceeded
				log.Error().Err(err).Str("filePath", path).Msg("Error deleting file after retries")
				break
			}
		}
	}
}

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

// AddToFileDeleteChan sends a file path to the fileDeleteChan with a non-blocking operation.
// If the channel is full, it logs a warning.
func AddToFileDeleteChan(path string) {
	select {
	case fileDeleteChan <- path:
		// Successfully added to the channel
	default:
		// Channel is full, log a warning
		log.Warn().Str("filePath", path).Msg("file deletion channel is full, unable to queue file for deletion")
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

// CloseDeleteChannels closes both the file and directory deletion channels
func CloseDeleteChannels() {
	close(fileDeleteChan)
	close(dirDeleteChan)
}
