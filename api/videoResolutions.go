package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

func VideoResolutions(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("videoFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", nil)
		return
	}

	videoPath := header.Header.Get("path")

	id := uuid.New().String()

	outputPath360 := fmt.Sprintf("media_docker_files/videos/%s/360", id)
	outputPath480 := fmt.Sprintf("media_docker_files/videos/%s/480", id)
	outputPath720 := fmt.Sprintf("media_docker_files/videos/%s/720", id)
	outputPath1080 := fmt.Sprintf("media_docker_files/videos/%s/1080", id)

	// Create the output directory
	if err := pkg.CreateDirs([]string{outputPath360, outputPath480, outputPath720, outputPath1080}); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error creating output directorys", err)
		go pkg.DeleteFile(videoPath)
		return
	}

	hlsPath360 := fmt.Sprintf("%s/index.m3u8", outputPath360)
	hlsPath480 := fmt.Sprintf("%s/index.m3u8", outputPath480)
	hlsPath720 := fmt.Sprintf("%s/index.m3u8", outputPath720)
	hlsPath1080 := fmt.Sprintf("%s/index.m3u8", outputPath1080)

	resolutions := []pkg.FFmpegConfig{
		{
			OutputPath: outputPath360,
			HlsPath:    hlsPath360,
			Resolution: 360,
		},
		{
			OutputPath: outputPath480,
			HlsPath:    hlsPath480,
			Resolution: 480,
		},
		{
			OutputPath: outputPath720,
			HlsPath:    hlsPath720,
			Resolution: 720,
		},
		{
			OutputPath: outputPath1080,
			HlsPath:    hlsPath1080,
			Resolution: 1080,
		},
	}

	err = pkg.ConvertVideoResolutions(videoPath, resolutions)

	if err != nil {
		helper.Response(w, http.StatusInternalServerError, "error while converting video", nil)
		go pkg.DeleteFile(videoPath)
		return
	}

	// Respond with success
	helper.Response(w, 201, "video uploaded successfully",
		map[string]any{
			"360":  fmt.Sprintf("http://localhost:7000/%s/index.m3u8", outputPath360),
			"480":  fmt.Sprintf("http://localhost:7000/%s/index.m3u8", outputPath480),
			"720":  fmt.Sprintf("http://localhost:7000/%s/index.m3u8", outputPath720),
			"1080": fmt.Sprintf("http://localhost:7000/%s/index.m3u8", outputPath1080),
		})

	go pkg.DeleteFile(videoPath)
}
