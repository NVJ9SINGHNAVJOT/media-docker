package pkg

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// LoadEnv loads environment variables from a given file, handling comments and preserving existing variables.
func LoadEnv(filePath string) error {
	// Use DirExist to check if the file exists and handle potential errors.
	exists, err := DirExist(filePath)
	if err != nil {
		return fmt.Errorf("error checking env file: %v", err)
	}
	if !exists {
		return fmt.Errorf("env file does not exist: %s", filePath)
	}

	// Open the specified environment variable file.
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening env file: %v", err)
	}
	defer file.Close() // Ensure the file is closed when done.

	scanner := bufio.NewScanner(file) // Create a scanner to read the file line by line.
	lineNumber := 0                   // Initialize a line number counter.

	for scanner.Scan() {
		lineNumber++                              // Increment line number for each line read.
		line := strings.TrimSpace(scanner.Text()) // Trim whitespace from the line.

		// Ignore empty lines and lines that are comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove inline comments after the actual environment variable.
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Split the line into key-value pairs by the first '=' character.
		keyValue := strings.SplitN(line, "=", 2)
		if len(keyValue) != 2 {
			return fmt.Errorf("invalid env line at line %d", lineNumber) // Return error with line number.
		}

		key := strings.TrimSpace(keyValue[0])   // Extract and trim the key.
		value := strings.TrimSpace(keyValue[1]) // Extract and trim the value.

		// Only set the environment variable if it doesn't already exist.
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value) // Set the environment variable.
		} else {
			log.Printf("Variable %s already exists, skipping...", key) // Log if the variable is already set.
		}
	}

	// Check for errors during scanning.
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading env file: %v", err) // Return any errors encountered.
	}

	return nil // Return nil if no errors occurred.
}
