package middleware

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
)

// valid file type
var validMimeTypes = map[string]bool{
	// image
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	// video
	"video/mp4":  true,
	"video/webm": true,
	"video/ogg":  true,
	// audio
	"audio/mp3":  true,
	"audio/mpeg": true,
	"audio/wav":  true,
}

// Change this to your desired folder path
var UploadFolder = "uploadStorage"

// max 1 GB file size allowed
const maxFileSize = 1024 * 1024 * 1000

func FileStorage(fileName string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Parse the form data
		err := r.ParseMultipartForm(maxFileSize)
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "error parsing form data", nil)
			return
		}

		// Check the number of files uploaded
		files := r.MultipartForm.File[fileName]
		if len(files) != 1 {
			helper.Response(w, http.StatusBadRequest, "only one file allowed", nil)
			return
		}

		file, header, err := r.FormFile(fileName)
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "error reading file", nil)
			return
		}

		if !validMimeTypes[header.Header.Get("Content-Type")] {
			helper.Response(w, http.StatusUnsupportedMediaType, "unsupported file type", nil)
			return
		}

		if header.Size > maxFileSize {
			helper.Response(w, http.StatusRequestEntityTooLarge, "file to large type", nil)
			return
		}

		defer file.Close()

		// Generate a unique filename with UUID
		fileExt := filepath.Ext(header.Filename)
		uuidFilename := fmt.Sprintf("%s-%s%s", strings.TrimSuffix(header.Filename, fileExt), uuid.New().String(), fileExt)
		filePath := filepath.Join(UploadFolder, uuidFilename)
		out, err := os.Create(filePath)

		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error creating file", nil)
			return
		}
		defer out.Close()

		// Copy the file content to the destination
		_, err = io.Copy(out, file)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error saving file", nil)
			return
		}

		header.Header.Add("path", filePath)
		next(w, r)
	}
}
