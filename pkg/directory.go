package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

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

// DirExist checks if the directory at the given dirPath exists.
// If the directory does not exist and createIfNotExist is true, it attempts to create the directory.
// The createIfNotExist argument is optional and defaults to false if not provided.
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
