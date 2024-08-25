package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

func Audio(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("audioFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err)
		return
	}

	audioPath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/audios", helper.Constants.MediaStorage)

	err = pkg.ConvertAudio(audioPath, outputPath, id)

	if err != nil {
		helper.Response(w, http.StatusInternalServerError, "error while converting audio", err)
		go pkg.DeleteFile(audioPath)
		return
	}

	// Respond with success
	audioUrl := fmt.Sprintf("http://localhost:7000/%s/%s.mp3", outputPath, id)
	helper.Response(w, 201, "audio uploaded successfully", map[string]any{"audioUrl": audioUrl})

	go pkg.DeleteFile(audioPath)
}
