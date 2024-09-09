package api

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/worker"
)

type videoRequest struct {
	Quality string `json:"quality" validate:"omitempty,customVideoQuality"`
}

func Video(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("videoFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}
	videoPath := header.Header.Get("path")

	var req videoRequest
	// Parse the JSON request and populate the VideoRequest struct.
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid data", err.Error())
		go pkg.DeleteFile(videoPath)
		return
	}

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, id)

	// Create the output directory
	if err := pkg.CreateDir(outputPath); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error creating output directory", err.Error())
		go pkg.DeleteFile(videoPath)
		return
	}

	var cmd *exec.Cmd

	if req.Quality != "" {
		quality, err := strconv.Atoi(req.Quality)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error while getting quality for video", err.Error())
			go pkg.DeleteFile(videoPath)
			return
		}
		cmd = pkg.ConvertVideo(videoPath, outputPath, quality)
	} else {
		cmd = pkg.ConvertVideo(videoPath, outputPath)
	}

	var executeError = false
	var wg sync.WaitGroup
	wg.Add(1)

	worker.AddInVideoChannel(cmd, &wg, &executeError)

	wg.Wait()

	if executeError {
		helper.Response(w, http.StatusInternalServerError, "error while converting video", nil)
		go pkg.DeleteFile(videoPath)
		return
	}

	// Respond with success
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.MDSenvs.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "video uploaded successfully", map[string]any{"fileUrl": videoUrl})

	go pkg.DeleteFile(videoPath)
}
