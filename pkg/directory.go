package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

// DirOrFileExist checks if a directory or file exists at the given path.
// It returns a boolean indicating the existence of the path and an error if any occurs during the check.
//
// Parameters:
// - path: A string representing the path to the directory or file to check.
//
// Returns:
// - bool: True if the directory or file exists, false if it does not exist.
// - error: An error indicating any issues encountered during the check (nil if no errors).
func DirOrFileExist(path string) (bool, error) {
	// Use os.Stat to get the file info for the specified path.
	_, err := os.Stat(path)
	if err != nil {
		// Check if the error indicates that the path does not exist.
		if os.IsNotExist(err) {
			// Return false with nil error if the path does not exist.
			return false, nil
		} else {
			// Return false and the error if a different error occurred.
			return false, err
		}
	}
	// Return true if the path exists without any error.
	return true, nil
}

// CreateDir creates a directory (and any necessary parent directories) at the given outputPath.
// It returns an error if the directory creation fails.
func CreateDir(outputPath string) error {
	return os.MkdirAll(outputPath, os.ModePerm) // Creates the directory with the full permissions.
}

// CreateDirs creates multiple directories given an array of output paths.
// It iterates through the outputPaths and calls CreateDir for each one.
// Returns an error if any directory creation fails.
func CreateDirs(outputPaths []string) error {
	for _, v := range outputPaths {
		err := CreateDir(v)
		if err != nil {
			return err // Return the error if directory creation fails.
		}
	}
	return nil // Return nil if all directories are created successfully.
}

// DirExist verifies if the directory at the specified dirPath exists.
// If the directory does not exist and createIfNotExist is set to true, it attempts to create the directory.
// The createIfNotExist parameter is optional and defaults to false when not provided.
//
// INFO: The function uses log.Fatal() if the directory does not exist,
// or if an error occurs while checking or creating the directory.
func DirExist(dirPath string, createIfNotExist ...bool) {
	// Default value for createIfNotExist is false
	shouldCreate := false

	// If a value is provided for createIfNotExist, use it
	if len(createIfNotExist) > 0 {
		shouldCreate = createIfNotExist[0]
	}

	_, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if shouldCreate {
				// Attempt to create the directory if createIfNotExist is true
				err := CreateDir(dirPath)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to create directory: " + dirPath)
				}
				log.Info().Msg(dirPath + " directory created successfully.")
			} else {
				log.Fatal().Msg(dirPath + " dir does not exist") // Log and terminate if directory does not exist and createIfNotExist is false.
			}
		} else {
			log.Fatal().Err(err).Msg("Error while checking " + dirPath + " dir") // Log and terminate if there is an error checking the directory.
		}
	}
}
