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
		// Parse the form data from the request with a file size limit for the file field.
		// The maximum file size is defined in helper.Constants.MaxFileSize for the given fileName.
		if err := r.ParseMultipartForm(helper.Constants.MaxFileSize[fileName]); err != nil {
			helper.Response(w, http.StatusBadRequest, "error parsing form data", err.Error())
			return
		}

		// Retrieve the file and its header information from the form data using the fileName field.
		// This reads the uploaded file from the request.
		file, header, err := r.FormFile(fileName)
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "error reading file - no file present", err.Error())
			return
		}
		// We will manually close the file later after processing.

		// Validate the file type by checking its MIME type using a custom function isValidFileType.
		// If the file type is not supported, respond with HTTP 415 Unsupported Media Type and close the file.
		if !isValidFileType(fileName, header.Header.Get("Content-Type")) {
			helper.Response(w, http.StatusUnsupportedMediaType, "unsupported "+fileName+" file type", nil)
			file.Close() // Manually close the file here since the function returns.
			return
		}

		// Check if the file size exceeds the allowed limit.
		// If the file is too large, respond with HTTP 413 Request Entity Too Large and close the file.
		if header.Size > helper.Constants.MaxFileSize[fileName] {
			helper.Response(w, http.StatusRequestEntityTooLarge, "file too large", nil)
			file.Close() // Manually close the file here since the function returns.
			return
		}

		// Create a map to store form values (non-file fields) from the form data.
		formData := make(map[string]string)

		// Iterate over the form values and add them to the formData map.
		// Only the first value for each key is considered.
		for key, values := range r.MultipartForm.Value {
			if len(values) > 0 {
				formData[key] = values[0]
			}
		}

		// If there are any form values, marshal them into JSON format.
		// Replace the request body with this JSON object for further use in the next handler.
		// Also update the ContentLength header to reflect the size of the new JSON body.
		if len(formData) > 0 {
			jsonData, err := json.Marshal(formData)
			if err != nil {
				helper.Response(w, http.StatusInternalServerError, "error marshalling JSON", nil)
				file.Close() // Manually close the file here since the function returns.
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(jsonData))
			r.ContentLength = int64(len(jsonData))
		} else {
			// If there are no form values, replace the body with an empty JSON object.
			r.Body = io.NopCloser(bytes.NewReader([]byte("{}")))
			r.ContentLength = int64(len("{}"))
		}

		// Generate a unique filename using the original file name and a UUID.
		// This ensures the uploaded file is saved with a unique name to prevent overwriting.
		fileExt := filepath.Ext(header.Filename)
		uuidFilename := fmt.Sprintf("%s-%s%s", strings.TrimSuffix(header.Filename, fileExt), uuid.New().String(), fileExt)
		filePath := filepath.Join(helper.Constants.UploadStorage, uuidFilename)

		// Create a new file on disk at the specified filePath.
		// This file will store the content of the uploaded file.
		out, err := os.Create(filePath)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error creating file", err.Error())
			file.Close() // Close the uploaded file before returning.
			return
		}
		// We will manually close the file and out streams later after copying the file content.

		// Copy the content of the uploaded file into the newly created file on disk.
		_, err = io.Copy(out, file)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error saving file", err.Error())
			file.Close() // Close the uploaded file.
			out.Close()  // Close the output file before returning.
			return
		}

		// Manually close the uploaded file and the output file after processing.
		file.Close()
		out.Close()

		// Add the path of the saved file to the response header, which can be used by the next handler.
		header.Header.Add("path", filePath)

		// Call the next handler in the chain to continue processing the request.
		// The uploaded file's path can be used by the next handler.
		next(w, r)
	}
}
