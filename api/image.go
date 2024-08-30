package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/worker"
)

func Image(w http.ResponseWriter, r *http.Request) {
	_, header, err := r.FormFile("imageFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}

	imagePath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/images/%s/jpeg", helper.Constants.MediaStorage, id)

	var executeError = false
	var wg sync.WaitGroup
	wg.Add(1)

	worker.AddInChannel(pkg.ConvertImage(imagePath, outputPath, "1"), &wg, &executeError)

	wg.Wait()

	if executeError {
		helper.Response(w, http.StatusInternalServerError, "error while converting image", nil)
		go pkg.DeleteFile(imagePath)
		return
	}

	// Respond with success
	helper.Response(w, http.StatusCreated, "image uploaded successfully", map[string]any{"fileUrl": outputPath})

	go pkg.DeleteFile(imagePath)
}
