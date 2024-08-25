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

// valid file types
var validFiles = map[string][]string{
	"image": {"image/jpeg", "image/jpg", "image/png"},
	"video": {"video/mp4", "video/webm", "video/ogg", "video/mkv"},
	"audio": {"audio/mp3", "audio/mpeg", "audio/wav"},
}

func isValidFileType(fileType, mimeType string) bool {
	allowedTypes, ok := validFiles[fileType]
	if !ok {
		return false // Invalid fileType
	}

	for _, allowedType := range allowedTypes {
		if allowedType == mimeType {
			return true // Valid fileType and mimeType
		}
	}

	return false // Valid fileType but invalid mimeType
}

func FileStorage(fileName string, fileType string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Parse the form data
		err := r.ParseMultipartForm(helper.Constants.MaxFileSize)
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "error parsing form data", err)
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
			helper.Response(w, http.StatusBadRequest, "error reading file", err)
			return
		}

		if !isValidFileType(fileType, header.Header.Get("Content-Type")) {
			helper.Response(w, http.StatusUnsupportedMediaType, "unsupported "+fileType+" file type", nil)
			return
		}

		if header.Size > helper.Constants.MaxFileSize {
			helper.Response(w, http.StatusRequestEntityTooLarge, "file to large type", nil)
			return
		}

		defer file.Close()

		// Generate a unique filename with UUID
		fileExt := filepath.Ext(header.Filename)
		uuidFilename := fmt.Sprintf("%s-%s%s", strings.TrimSuffix(header.Filename, fileExt), uuid.New().String(), fileExt)
		filePath := filepath.Join(helper.Constants.UploadStorage, uuidFilename)
		out, err := os.Create(filePath)

		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error creating file", err)
			return
		}
		defer out.Close()

		// Copy the file content to the destination
		_, err = io.Copy(out, file)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error saving file", err)
			return
		}

		header.Header.Add("path", filePath)
		next(w, r)
	}
}
