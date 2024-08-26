package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

func Video(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("videoFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}

	videoPath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, id)

	// Create the output directory
	if err := pkg.CreateDir(outputPath); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error creating output directory", err.Error())
		go pkg.DeleteFile(videoPath)
		return
	}

	err = pkg.ConvertVideo(videoPath, outputPath)

	if err != nil {
		helper.Response(w, http.StatusInternalServerError, "error while converting video", err.Error())
		go pkg.DeleteFile(videoPath)
		return
	}

	// Respond with success
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.Envs.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "video uploaded successfully", map[string]any{"fileUrl": videoUrl})

	go pkg.DeleteFile(videoPath)
}
