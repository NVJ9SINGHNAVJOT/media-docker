package helper

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// FileConfig holds the configuration for a specific file category,
// including the allowed MIME types and the maximum allowed size for uploads.
type FileConfig struct {
	AllowedTypes []string // List of allowed MIME types for this file category
	MaxSize      int64    // Maximum allowed file size in bytes
	// AllowAnyType disables the AllowedTypes check for this category.
	// Only the "other" category sets this, since it exists specifically to
	// accept arbitrary files. Size limits still apply.
	//
	// CAUTION: files in an AllowAnyType category are attacker-controlled content
	// served from the media-docker-client origin. They must always be served with
	// Content-Disposition: attachment and X-Content-Type-Options: nosniff,
	// otherwise an uploaded HTML file becomes stored XSS.
	AllowAnyType bool
	// DefaultExt is used when the extension cannot be derived from the uploaded
	// file name (see SanitizeExt).
	DefaultExt string
}

// constConfig holds the overall configuration for file uploads,
// including storage locations and file category settings.
type constConfig struct {
	UploadStorage string // Directory for staging chunked uploads before they are merged
	MediaStorage  string // Directory for storing media files (publicly served)
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

	// Categories that accept arbitrary uploads skip the MIME allowlist entirely.
	if fileConfig.AllowAnyType {
		return true
	}

	// Check if the provided MIME type is in the list of allowed types
	return slices.Contains(fileConfig.AllowedTypes, mimeType) // Valid file category, but the MIME type is not allowed
}

// extPattern matches an acceptable file extension: alphanumeric, 1-8 characters.
// Anything outside this is rejected rather than sanitized, so that a hostile file
// name can never contribute path separators or dot segments to a stored path.
var extPattern = regexp.MustCompile(`^[A-Za-z0-9]{1,8}$`)

// SanitizeExt determines the extension to store a file under.
//
// It prefers the extension of the client-supplied file name, because deriving it
// from the MIME subtype breaks for real-world document types: the subtype of a
// .docx is "vnd.openxmlformats-officedocument.wordprocessingml.document", which is
// neither a valid nor a useful extension.
//
// The MIME subtype is used as a fallback only when it is itself a plausible
// extension (e.g. "video/mp4" -> "mp4"), and the category default is used when
// neither source yields anything usable.
//
// Parameters:
//   - fileCategory: the file category (e.g., "video", "document")
//   - fileName: the original file name supplied by the client (may be empty)
//   - mimeType: the Content-Type reported for the uploaded part
//
// Returns the extension without a leading dot, always lowercase.
func (c *constConfig) SanitizeExt(fileCategory, fileName, mimeType string) string {
	// Prefer the extension from the original file name.
	if ext := strings.TrimPrefix(filepath.Ext(fileName), "."); extPattern.MatchString(ext) {
		return strings.ToLower(ext)
	}

	// Fall back to the MIME subtype, but only when it looks like an extension.
	if _, subtype, found := strings.Cut(mimeType, "/"); found && extPattern.MatchString(subtype) {
		return strings.ToLower(subtype)
	}

	// Last resort: the category default.
	return c.Files[fileCategory].DefaultExt
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
			DefaultExt:   "jpeg",
		},
		"video": {
			AllowedTypes: []string{"video/mp4", "video/webm", "video/ogg", "video/mkv"}, // Allowed video MIME types
			MaxSize:      1024 * 1024 * 1000,                                            // Maximum size for video uploads (1 GB)
			DefaultExt:   "mp4",
		},
		"audio": {
			AllowedTypes: []string{"audio/mp3", "audio/mpeg", "audio/wav"}, // Allowed audio MIME types
			MaxSize:      1024 * 1024 * 50,                                 // Maximum size for audio uploads (50 MB)
			DefaultExt:   "mp3",
		},
		// Documents are stored and served as-is; no conversion is performed.
		"document": {
			AllowedTypes: []string{
				"application/pdf",
				"application/msword",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				"application/vnd.ms-excel",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/vnd.ms-powerpoint",
				"application/vnd.openxmlformats-officedocument.presentationml.presentation",
				"application/vnd.oasis.opendocument.text",
				"text/plain",
				"text/csv",
			},
			MaxSize:    1024 * 1024 * 100, // Maximum size for document uploads (100 MB)
			DefaultExt: "bin",
		},
		// "other" accepts any file type; it exists for uploads that do not fit the
		// categories above. Stored and served as-is, always as an attachment.
		"other": {
			AllowAnyType: true,
			MaxSize:      1024 * 1024 * 200, // Maximum size for other uploads (200 MB)
			DefaultExt:   "bin",
		},
	},
}
