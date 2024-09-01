package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/worker"
)

func VideoResolutions(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("videoFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", nil)
		return
	}

	videoPath := header.Header.Get("path")

	id := uuid.New().String()

	outputPath360 := fmt.Sprintf("%s/videos/%s/360", helper.Constants.MediaStorage, id)
	outputPath480 := fmt.Sprintf("%s/videos/%s/480", helper.Constants.MediaStorage, id)
	outputPath720 := fmt.Sprintf("%s/videos/%s/720", helper.Constants.MediaStorage, id)
	outputPath1080 := fmt.Sprintf("%s/videos/%s/1080", helper.Constants.MediaStorage, id)

	// Create the output directory
	if err := pkg.CreateDirs([]string{outputPath360, outputPath480, outputPath720, outputPath1080}); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error creating output directorys", err.Error())
		go pkg.DeleteFile(videoPath)
		return
	}

	var executeError = false
	var wg sync.WaitGroup
	wg.Add(4)

	worker.AddInVideoResolutionChannel(pkg.ConvertVideoResolution(videoPath, outputPath360, "360"), &wg, &executeError)
	worker.AddInVideoResolutionChannel(pkg.ConvertVideoResolution(videoPath, outputPath480, "480"), &wg, &executeError)
	worker.AddInVideoResolutionChannel(pkg.ConvertVideoResolution(videoPath, outputPath720, "720"), &wg, &executeError)
	worker.AddInVideoResolutionChannel(pkg.ConvertVideoResolution(videoPath, outputPath1080, "1080"), &wg, &executeError)

	wg.Wait()

	if executeError {
		helper.Response(w, http.StatusInternalServerError, "error while converting video resolutions", nil)
		go pkg.DeleteFile(videoPath)
		return
	}

	// Respond with success
	helper.Response(w, http.StatusCreated, "video uploaded successfully",
		map[string]any{
			"360":  fmt.Sprintf("%s/%s/index.m3u8", config.MDSenvs.BASE_URL, outputPath360),
			"480":  fmt.Sprintf("%s/%s/index.m3u8", config.MDSenvs.BASE_URL, outputPath480),
			"720":  fmt.Sprintf("%s/%s/index.m3u8", config.MDSenvs.BASE_URL, outputPath720),
			"1080": fmt.Sprintf("%s/%s/index.m3u8", config.MDSenvs.BASE_URL, outputPath1080),
		})

	go pkg.DeleteFile(videoPath)
}
