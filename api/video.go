package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

func Video(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("videoFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err)
		return
	}

	videoPath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("media_docker_files/videos/%s", id)
	hlsPath := fmt.Sprintf("%s/index.m3u8", outputPath)

	// Create the output directory
	if err := pkg.CreateDir(outputPath); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error creating output directory", err)
		go pkg.DeleteFile(videoPath)
		return
	}

	err = pkg.ConvertVideo(videoPath, outputPath, hlsPath)

	if err != nil {
		helper.Response(w, http.StatusInternalServerError, "error while converting video", err)
		go pkg.DeleteFile(videoPath)
		return
	}

	// Respond with success
	videoURL := fmt.Sprintf("http://localhost:7000/%s/index.m3u8", outputPath)
	helper.Response(w, 201, "video uploaded successfully", map[string]any{"videoUrl": videoURL})

	go pkg.DeleteFile(videoPath)
}
