package api

import (
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/rs/zerolog/log"
)

// FileStorage handles single-request uploads for files that fit in one chunk,
// validates the input data, and writes the file straight into media storage.
//
// Unlike the chunked path there is nothing to stage: the file arrives whole, so
// it is written directly to its final location and is resolvable as soon as the
// request completes.
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

	contentType := header.Header.Get("Content-Type")

	// Validate the file type using a custom validation function.
	if !helper.Constants.IsValidFileType(fileType, contentType) {
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

	status := fileStatus{
		Type:     fileType,
		ID:       uuid.New().String(),
		FileName: r.FormValue("fileName"),
	}

	// Write the uploaded bytes into the asset directory and record its metadata.
	err = finalizeUpload(status, contentType, func(finalPath string) error {
		out, err := os.Create(finalPath)
		if err != nil {
			return err
		}

		if _, err = io.Copy(out, file); err != nil {
			out.Close()
			return err
		}

		return out.Close()
	})

	// Close the uploaded file to release system resources after the file is saved.
	if closeErr := file.Close(); closeErr != nil {
		log.Warn().Err(closeErr).Msgf("Warning: Could not close uploaded file for asset: %s", status.ID)
	}

	if err != nil {
		// Respond with a 500 Internal Server Error if there's an issue saving the file.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error saving file", err)
		return
	}

	// Respond with a 200 OK, indicating that the file was successfully uploaded.
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, "file uploaded successfully", map[string]string{"id": status.ID})
}
