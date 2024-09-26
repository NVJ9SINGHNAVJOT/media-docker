package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

// Channels for file and directory deletion
var fileDeleteChan = make(chan string)
var dirDeleteChan = make(chan string)

// DeleteFileWorker listens on the fileDeleteChan and deletes files as requested
func DeleteFileWorker() {
	for path := range fileDeleteChan {
		err := os.Remove(path)
		if err != nil {
			log.Error().Err(err).Str("filePath", path).Msg("error deleting file")
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
