package api

import (
	"fmt"
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
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

	// Respond with success
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.MDSenvs.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "video uploaded successfully", map[string]any{"fileUrl": videoUrl})

	go pkg.DeleteFile(videoPath)
}
