package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
)

// validFiles defines allowed MIME types for different file categories.
// Key is the file category name and value is a slice of allowed MIME types for that category.
var validFiles = map[string][]string{
	"imageFile": {"image/jpeg", "image/jpg", "image/png"},
	"videoFile": {"video/mp4", "video/webm", "video/ogg", "video/mkv"},
	"audioFile": {"audio/mp3", "audio/mpeg", "audio/wav"},
}

// isValidFileType checks if the provided MIME type is valid for the given file category.
// fileName: the name of the file form field
// mimeType: the MIME type of the uploaded file
// Returns true if the MIME type is valid for the file category, otherwise false.
func isValidFileType(fileName, mimeType string) bool {
	// Retrieve allowed MIME types for the given file category
	allowedTypes, ok := validFiles[fileName]
	if !ok {
		return false // Invalid file category
	}

	// Check if the provided MIME type is in the list of allowed types
	for _, allowedType := range allowedTypes {
		if allowedType == mimeType {
			return true // Valid MIME type for the file category
		}
	}

	return false // Valid file category but invalid MIME type
}

// FileStorage handles file uploads and processes form data.
// fileName: the name of the file form field
// next: a handler function to call after processing the file upload
// Returns an http.HandlerFunc that processes the file upload and calls the next handler.
func FileStorage(fileName string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the form data, with a maximum file size defined for the file field
		if err := r.ParseMultipartForm(helper.Constants.MaxFileSize[fileName]); err != nil {
			helper.Response(w, http.StatusBadRequest, "error parsing form data", err.Error())
			return
		}

		// Retrieve the file from the form data
		file, header, err := r.FormFile(fileName)
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "error reading file - no file present", err.Error())
			return
		}

		// Validate the file type based on MIME type
		if !isValidFileType(fileName, header.Header.Get("Content-Type")) {
			helper.Response(w, http.StatusUnsupportedMediaType, "unsupported "+fileName+" file type", nil)
			return
		}

		// Check if the file size exceeds the maximum allowed size
		if header.Size > helper.Constants.MaxFileSize[fileName] {
			helper.Response(w, http.StatusRequestEntityTooLarge, "file too large", nil)
			return
		}

		// Create a map to store form values
		formData := make(map[string]string)

		// Iterate over form values and add them to the map
		for key, values := range r.MultipartForm.Value {
			if len(values) > 0 {
				formData[key] = values[0]
			}
		}

		// Convert the formData map to JSON if it contains any values
		if len(formData) > 0 {
			jsonData, err := json.Marshal(formData)
			if err != nil {
				helper.Response(w, http.StatusInternalServerError, "error marshalling JSON", nil)
				return
			}

			// Replace the request body with the JSON object
			r.Body = io.NopCloser(bytes.NewReader(jsonData))
			r.ContentLength = int64(len(jsonData))
		} else {
			// Set an empty JSON body if there are no form values
			r.Body = io.NopCloser(bytes.NewReader([]byte("{}")))
			r.ContentLength = int64(len("{}"))
		}

		defer file.Close()

		// Generate a unique filename using UUID
		fileExt := filepath.Ext(header.Filename)
		uuidFilename := fmt.Sprintf("%s-%s%s", strings.TrimSuffix(header.Filename, fileExt), uuid.New().String(), fileExt)
		filePath := filepath.Join(helper.Constants.UploadStorage, uuidFilename)

		// Create the file on disk
		out, err := os.Create(filePath)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error creating file", err.Error())
			return
		}
		defer out.Close()

		// Copy the file content to the new file
		_, err = io.Copy(out, file)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error saving file", err.Error())
			return
		}

		// Add the file path to the response header for further use
		header.Header.Add("path", filePath)

		// Call the next handler function
		next(w, r)
	}
}
