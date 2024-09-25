package pkg

// Channels for file and directory deletion
var fileDeleteChan = make(chan string)
var dirDeleteChan = make(chan string)

// DeleteFileWorker listens on the fileDeleteChan and deletes files as requested
func DeleteFileWorker() {
	for path := range fileDeleteChan {
		// Call the DeleteFile function
		DeleteFile(path)
	}
}

// DeleteDirWorker listens on the dirDeleteChan and deletes directories as requested
func DeleteDirWorker() {
	for path := range dirDeleteChan {
		// Call the DeleteDir function
		DeleteDir(path)
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
