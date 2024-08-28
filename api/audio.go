package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

func Audio(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("audioFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}

	audioPath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/audios", helper.Constants.MediaStorage)

	err = pkg.ConvertAudio(audioPath, outputPath, id)

	if err != nil {
		helper.Response(w, http.StatusInternalServerError, "error while converting audio", err.Error())
		go pkg.DeleteFile(audioPath)
		return
	}

	// Respond with success
	audioUrl := fmt.Sprintf("%s/%s/%s.mp3", config.MDSenvs.BASE_URL, outputPath, id)
	helper.Response(w, http.StatusCreated, "audio uploaded successfully", map[string]any{"fileUrl": audioUrl})

	go pkg.DeleteFile(audioPath)
}
