package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	serverKafka "github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

type VideoResolutionMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New ID for video URL
}

func VideoResolutions(w http.ResponseWriter, r *http.Request) {
	_, header, err := r.FormFile("videoFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}

	videoPath := header.Header.Get("path")
	id := uuid.New().String()

	// Create the VideoResolutionMessage struct to be passed to Kafka
	message := VideoResolutionMessage{
		FilePath: videoPath,
		NewId:    id,
	}

	// Create a channel of size 1 to store the Kafka response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the id as the key
	serverKafka.VideoResolutionRequestMap.Store(id, responseChannel)

	// Pass the struct to the Kafka producer
	if err := serverKafka.KafkaProducer.Produce("videoResolution", message); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err.Error())
		serverKafka.VideoResolutionRequestMap.Delete(videoPath)
		go pkg.DeleteFile(videoPath)
		return
	}

	// Wait for the response from the Kafka processor
	responseSuccess := <-responseChannel
	serverKafka.VideoResolutionRequestMap.Delete(videoPath)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "video resolution conversion failed", nil)
		go pkg.DeleteFile(videoPath)
		return
	}

	// Respond with success
	helper.Response(w, http.StatusCreated, "video uploaded successfully",
		map[string]any{
			"360":  fmt.Sprintf("%s/%s/videos/%s/360", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
			"480":  fmt.Sprintf("%s/%s/videos/%s/480", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
			"720":  fmt.Sprintf("%s/%s/videos/%s/720", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
			"1080": fmt.Sprintf("%s/%s/videos/%s/1080", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
		})

	go pkg.DeleteFile(videoPath)
}
