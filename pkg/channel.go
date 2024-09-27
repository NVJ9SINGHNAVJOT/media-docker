package pkg

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Channels for file and directory deletion
var (
	fileDeleteChan = make(chan string)
	dirDeleteChan  = make(chan string)
)

const (
	retries  = 2
	waitTime = 2 * time.Second // Constant 2 seconds delay between retries
)

// DeleteFileWorker listens on the fileDeleteChan and deletes files as requested with retries
//
// This function gives a total of 3 attempts for deleting the file. If it fails after 2 attempts,
// the 3rd attempt will log an error. The retry wait time is fixed at 2 seconds.
func DeleteFileWorker() {
	for path := range fileDeleteChan {

		for i := 0; i <= retries; i++ {
			err := os.Remove(path)
			if err == nil {
				break // Successfully deleted, no need to retry
			}

			// If the error is because the file is in use, retry
			if i < retries && strings.Contains(err.Error(), "The process cannot access the file because it is being used by another process.") {
				log.Warn().Msgf("Path: %s, Attempt %d to delete file failed because it is in use. Retrying in 2 seconds...", path, i+1)
				time.Sleep(waitTime) // Constant 2-second wait before retrying
			} else {
				// Log error if it's not a file-in-use error or max retries exceeded
				log.Error().Err(err).Str("filePath", path).Msg("error deleting file after retries")
				break
			}
		}
	}
}

// DeleteDirWorker listens on the dirDeleteChan and deletes directories as requested
func DeleteDirWorker() {
	for path := range dirDeleteChan {
		err := os.RemoveAll(path)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("error deleting directory")
		}
	}
}

// addToFileDeleteChan sends a file path to the fileDeleteChan
func AddToFileDeleteChan(path string) {
	fileDeleteChan <- path
}

// addToDirDeleteChan sends a directory path to the dirDeleteChan
func AddToDirDeleteChan(path string) {
	dirDeleteChan <- path
}

// CloseDeleteChannels closes the deletion channels
func CloseDeleteChannels() {
	close(fileDeleteChan)
	close(dirDeleteChan)
}

// CloseDeleteFileChannel closes the file deletion channel
func CloseDeleteFileChannel() {
	close(fileDeleteChan)
}
