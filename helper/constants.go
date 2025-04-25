package helper

import "slices"

// FileConfig holds the configuration for a specific file category,
// including the allowed MIME types and the maximum allowed size for uploads.
type FileConfig struct {
	AllowedTypes []string // List of allowed MIME types for this file category
	MaxSize      int64    // Maximum allowed file size in bytes
}

// constConfig holds the overall configuration for file uploads,
// including storage locations and file category settings.
type constConfig struct {
	UploadStorage string // Directory for storing uploaded files
	MediaStorage  string // Directory for storing media files
	// MaxChunkSize defines the maximum size for each file chunk,
	// set to 2 MB (2 * 1024 * 1024 bytes), in accordance with
	// the MediaDocker module specifications.
	MaxChunkSize int64                 // Max chunk size allowed in form data
	Files        map[string]FileConfig // Map of file categories to their respective configurations
}

// IsValidFileType checks if the provided MIME type is valid for the given file category.
//
// Parameters:
// - fileCategory: the name of the file category (e.g., "image", "video")
// - mimeType: the MIME type of the uploaded file
//
// Returns:
// - true if the MIME type is valid for the file category, otherwise false.
func (c *constConfig) IsValidFileType(fileCategory, mimeType string) bool {
	// Retrieve the file configuration for the specified file category
	fileConfig, ok := c.Files[fileCategory]
	if !ok {
		return false // Invalid file category; return false
	}

	// Check if the provided MIME type is in the list of allowed types
	return slices.Contains(fileConfig.AllowedTypes, mimeType) // Valid file category, but the MIME type is not allowed
}

// NOTE: do not change these values, project will break
var Constants = &constConfig{
	UploadStorage: "uploadStorage",      // Path to the directory where files will be uploaded
	MediaStorage:  "media_docker_files", // Path to the directory for media storage
	// maxChunkSize defines the maximum size for each file chunk,
	// set to 2 MB (2 * 1024 * 1024 bytes), in accordance with
	// the MediaDocker module specifications.
	MaxChunkSize: 1024 * 1024 * 2, // 2 MB
	Files: map[string]FileConfig{ // Configuration for different file types
		"image": {
			AllowedTypes: []string{"image/jpeg", "image/jpg", "image/png"}, // Allowed image MIME types
			MaxSize:      1024 * 1024 * 50,                                 // Maximum size for image uploads (50 MB)
		},
		"video": {
			AllowedTypes: []string{"video/mp4", "video/webm", "video/ogg", "video/mkv"}, // Allowed video MIME types
			MaxSize:      1024 * 1024 * 1000,                                            // Maximum size for video uploads (1 GB)
		},
		"audio": {
			AllowedTypes: []string{"audio/mp3", "audio/mpeg", "audio/wav"}, // Allowed audio MIME types
			MaxSize:      1024 * 1024 * 50,                                 // Maximum size for audio uploads (50 MB)
		},
	},
}
