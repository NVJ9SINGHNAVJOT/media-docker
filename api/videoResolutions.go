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

// VideoResolutionsMessage represents the structure of the message sent to Kafka for video resolution processing.
type VideoResolutionsMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New unique identifier for the video file URL
}

// VideoResolutions handles video file upload requests and sends processing messages to Kafka for resolution conversion.
func VideoResolutions(w http.ResponseWriter, r *http.Request) {
	// Read the video file from the request
	_, header, err := r.FormFile("videoFile")
	if err != nil {
		// Respond with an error if the file cannot be read
		helper.Response(w, http.StatusBadRequest, "error reading file", err)
		return
	}

	videoPath := header.Header.Get("path") // Get the file path from the header
	id := uuid.New().String()              // Generate a new UUID for the video file

	// Create the VideoResolutionsMessage struct to be passed to Kafka
	message := VideoResolutionsMessage{
		FilePath: videoPath, // Set the file path
		NewId:    id,        // Set the new ID for the file URL
	}

	// Pass the struct to the Kafka producer
	if err := serverKafka.KafkaProducer.Produce("video-resolutions", message); err != nil {
		pkg.AddToFileDeleteChan(videoPath) // Add to deletion channel on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing URLs for different video resolutions
	helper.Response(w, http.StatusCreated, "video uploaded successfully",
		map[string]any{
			"id": id,
			"fileUrls": map[string]string{
				"360":  fmt.Sprintf("%s/%s/videos/%s/360/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
				"480":  fmt.Sprintf("%s/%s/videos/%s/480/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
				"720":  fmt.Sprintf("%s/%s/videos/%s/720/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
				"1080": fmt.Sprintf("%s/%s/videos/%s/1080/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
			},
		})
}
