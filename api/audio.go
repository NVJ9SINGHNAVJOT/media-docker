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

func Audio(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("audioFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}

	audioPath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, id)

	var executeError = false
	var wg sync.WaitGroup
	wg.Add(1)

	worker.AddInAudioChannel(pkg.ConvertAudio(audioPath, outputPath), &wg, &executeError)

	wg.Wait()

	if executeError {
		helper.Response(w, http.StatusInternalServerError, "error while converting audio", nil)
		go pkg.DeleteFile(audioPath)
		return
	}

	// Respond with success
	helper.Response(w, http.StatusCreated, "audio uploaded successfully",
		map[string]any{"fileUrl": fmt.Sprintf("%s/%s", config.MDSenvs.BASE_URL, outputPath)})

	go pkg.DeleteFile(audioPath)
}
