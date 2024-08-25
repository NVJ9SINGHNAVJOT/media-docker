package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

func Image(w http.ResponseWriter, r *http.Request) {
	_, header, err := r.FormFile("imageFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}

	imagePath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/images", helper.Constants.MediaStorage)

	err = pkg.ConvertImage(imagePath, outputPath, id, 1)

	if err != nil {
		helper.Response(w, http.StatusInternalServerError, "error while converting image", err.Error())
		go pkg.DeleteFile(imagePath)
		return
	}

	// Respond with success
	imageUrl := fmt.Sprintf("http://localhost:7000/%s/%s.jpeg", outputPath, id)
	helper.Response(w, 201, "image uploaded successfully", map[string]any{"imageUrl": imageUrl})

	go pkg.DeleteFile(imagePath)
}
