package process

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

// removeFile attempts to remove the file at the given path.
// It will make up to 3 attempts to delete the file, logging warnings
// on failure and retrying after a 2-second delay between attempts.
// If the file cannot be removed after 3 attempts, it logs an error.
func removeFile(workerName, path string) {
	for i := 1; i <= 3; i++ {
		// Attempt to remove the file at the specified path.
		err := os.Remove(path)
		if err == nil {
			// If the file is successfully deleted, return immediately.
			return
		}

		// If the attempt failed, check if more retries are available.
		if i < 3 {
			// Log a warning with the worker name, file path, and retry count.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d to delete file %s failed. Retrying in 2 seconds...", i, path)

			// Wait for 2 seconds before retrying.
			time.Sleep(2 * time.Second)
		} else {
			// If all retries are exhausted, log an error with the details.
			log.Error().
				Err(err).
				Str("worker", workerName).
				Str("filePath", path).
				Msg("Failed to delete file after 3 attempts")
		}
	}
}

// RemoveDir attempts to remove the specified directory.
// It makes up to 3 attempts to delete the directory, logging warnings on failure
// and retrying after a 2-second delay between attempts. If the directory cannot
// be removed after 3 attempts, it logs an error.
func RemoveDir(workerName, outputPath string) {
	for i := 1; i <= 3; i++ {
		// Attempt to remove the directory at the specified path.
		err := os.Remove(outputPath)
		if err == nil {
			// If the directory is successfully deleted, return immediately.
			return
		}

		// If the attempt failed, check if more retries are available.
		if i < 3 {
			// Log a warning with the worker name, directory path, and retry count.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d to delete directory %s failed. Retrying in 2 seconds...", i, outputPath)

			// Wait for 2 seconds before retrying.
			time.Sleep(2 * time.Second)
		} else {
			// If all retries are exhausted, log an error with the details.
			log.Error().
				Err(err).
				Str("worker", workerName).
				Str("dirPath", outputPath).
				Msg("Failed to delete directory after 3 attempts")
		}
	}
}

// createOutputDirectory attempts to create the specified directory with a maximum of 3 retries.
// It logs an error if all attempts fail and returns the error.
func createOutputDirectory(workerName, outputPath string) error {
	// Retry the directory creation process for a maximum of 3 attempts.
	for attempt := 1; attempt <= 3; attempt++ {
		// Attempt to create the directory.
		err := pkg.CreateDir(outputPath)
		if err == nil {
			// If successful, return immediately.
			return nil
		}

		// If the last attempt fails, log the error and return it.
		if attempt == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Failed to create output directory %s after 3 attempts.", outputPath)
			return fmt.Errorf("failed to create directory after 3 attempts: %s, %v", outputPath, err)
		}

		// Log a warning and retry if the directory creation fails before the final attempt.
		log.Warn().
			Err(err).
			Str("worker", workerName).
			Msgf("Attempt %d to create output directory %s failed. Retrying in 1 second...", attempt, outputPath)

		// Wait for 1 second before retrying.
		time.Sleep(1 * time.Second)
	}

	// This point will not be reached, since the function either returns success or an error after 3 attempts.
	return nil
}

// cleanupOutputDirectory attempts to remove all files and subdirectories within the specified outputPath.
// It does not remove the outputPath itself. The cleanup process is retried up to 3 times in case of failure.
func cleanupOutputDirectory(workerName, outputPath string) error {
	// Retry the cleanup process for a maximum of 3 attempts.
	for attempt := 1; attempt <= 3; attempt++ {
		// Walk through the directory to delete files and subdirectories.
		err := filepath.Walk(outputPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err // Return immediately if an error occurs while walking the directory.
			}

			// Skip the outputPath directory itself but continue cleaning its contents.
			if path == outputPath {
				return nil // Continue walking without deleting the root outputPath.
			}

			// Attempt to remove the file or directory.
			if info.IsDir() {
				// Remove the directory and its contents.
				err = os.RemoveAll(path)
				if err != nil {
					return fmt.Errorf("failed to remove directory %s: %v", path, err)
				}
			} else {
				// Remove the file.
				err = os.Remove(path)
				if err != nil {
					return fmt.Errorf("failed to remove file %s: %v", path, err)
				}
			}
			return nil
		})

		if err == nil {
			// If no error occurred, the cleanup was successful. Return immediately.
			return nil
		}

		// Log a warning and retry if an error occurs, unless it's the final attempt.
		if attempt == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Failed to clean up output directory %s after 3 attempts.", outputPath)
			return fmt.Errorf("failed to clean up output directory after 3 attempts: %s, %v", outputPath, err)
		}

		// Log a warning and wait for 1 second before retrying.
		log.Warn().
			Err(err).
			Str("worker", workerName).
			Msgf("Attempt %d to clean up output directory %s failed. Retrying in 1 second...", attempt, outputPath)

		time.Sleep(1 * time.Second)
	}

	// This point will not be reached, since the function either returns success or an error after 3 attempts.
	return nil
}
