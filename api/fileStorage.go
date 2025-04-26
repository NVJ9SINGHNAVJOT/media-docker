package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/rs/zerolog/log"
)

// FileStorage handles file upload requests, validates the input data,
// and saves the uploaded file to disk.
func FileStorage(w http.ResponseWriter, r *http.Request) {
	fileType := r.FormValue("type")

	// Check if the specified file type exists in the helper's constants.
	_, exist := helper.Constants.Files[fileType]
	if !exist {
		// Respond with a 400 Bad Request if the file type is invalid.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid type in form data", nil)
		return
	}

	fileName := fileType + "File"

	// Parse the form data with a maximum allowed file size specified by helper.Constants.MaxChunkSize.
	if err := r.ParseMultipartForm(helper.Constants.MaxChunkSize); err != nil {
		// Respond with a 400 Bad Request if there's an error parsing the form data.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "error parsing form data", err)
		return
	}

	// Retrieve the uploaded file and its header information from the form data.
	file, header, err := r.FormFile(fileName)
	if err != nil {
		// Respond with a 400 Bad Request if no file is present in the request.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "error reading file - no file present", err)
		return
	}

	// Validate the file type using a custom validation function.
	if !helper.Constants.IsValidFileType(fileType, header.Header.Get("Content-Type")) {
		// Respond with a 415 Unsupported Media Type if the file type is invalid.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusUnsupportedMediaType, "unsupported "+fileName+" file type", nil)
		file.Close() // Close the uploaded file before returning to free resources.
		return
	}

	// Check if the uploaded file size exceeds the maximum allowed size.
	if header.Size > helper.Constants.MaxChunkSize {
		// Respond with a 413 Request Entity Too Large if the file size exceeds the limit.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusRequestEntityTooLarge, "file too large", nil)
		file.Close() // Close the uploaded file before returning.
		return
	}

	// Generate a unique filename using a UUID and the file's content type to avoid name collisions.
	uuidFilename := fmt.Sprintf("%s.%s",
		uuid.New().String(),
		strings.Split(header.Header.Get("Content-Type"), "/")[1],
	)
	filePath := filepath.Join(helper.Constants.UploadStorage, uuidFilename)

	// Create a new file on disk at the specified path to store the uploaded file content.
	out, err := os.Create(filePath)
	if err != nil {
		// Respond with a 500 Internal Server Error if there's an issue creating the file.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error creating file", err)
		file.Close() // Close the uploaded file before returning.
		return
	}
	// Both the input (uploaded file) and output (new file) will be closed later.

	// Copy the content of the uploaded file to the newly created file on disk.
	_, err = io.Copy(out, file)
	if err != nil {
		// Respond with a 500 Internal Server Error if there's an issue saving the file.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error saving file", err)
		file.Close() // Close the uploaded file.
		out.Close()  // Close the output file before returning.
		return
	}

	// Close the uploaded file and output file to release system resources after the file is saved.
	if err := file.Close(); err != nil {
		log.Warn().Err(err).Msgf("Warning: Could not close uploaded file: %s", filePath)
	}
	if err := out.Close(); err != nil {
		log.Warn().Err(err).Msgf("Warning: Could not close output file: %s", filePath)
	}

	// Respond with a 200 OK, indicating that the file was successfully uploaded.
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, "file uploaded successfully", map[string]string{"uuidFilename": uuidFilename})
}
