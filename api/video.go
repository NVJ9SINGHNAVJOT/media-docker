package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
)

// videoRequest represents the structure of the request for video upload.
type videoRequest struct {
	Quality string `json:"quality" validate:"omitempty,customVideoQuality"` // Optional quality parameter
}

// Video handles video upload requests and sends processing messages to Kafka.
func Video(w http.ResponseWriter, r *http.Request) {
	// Read the video file from the request
	_, header, err := r.FormFile("videoFile")
	if err != nil {
		// Respond with an error if the file cannot be read
		helper.Response(w, http.StatusBadRequest, "error reading file", err)
		return
	}
	videoPath := header.Header.Get("path") // Get the file path from the header

	var req videoRequest
	// Parse the JSON request and populate the VideoRequest struct
	if err := helper.ValidateRequest(r, &req); err != nil {
		pkg.AddToFileDeleteChan(videoPath) // Add to deletion channel on error
		helper.Response(w, http.StatusBadRequest, "invalid data", err)
		return
	}

	// Initialize quality as nil for optional use
	var quality *int = nil
	// Convert quality string to an integer if provided
	if req.Quality != "" {
		q, err := strconv.Atoi(req.Quality) // Convert string to int
		if err != nil {
			pkg.AddToFileDeleteChan(videoPath) // Add to deletion channel on error
			helper.Response(w, http.StatusBadRequest, "invalid quality value", err)
			return
		}
		quality = &q // Set quality as a pointer to the integer value
	}

	id := uuid.New().String()                                                    // Generate a new UUID for the video
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, id) // Define the output path for the video

	// Create the VideoMessage struct to be passed to Kafka
	message := topics.VideoMessage{
		FilePath: videoPath, // Set the file path
		NewId:    id,        // Set the new ID
		Quality:  quality,   // Set the optional quality (can be nil)
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("video", message); err != nil {
		pkg.AddToFileDeleteChan(videoPath) // Add to deletion channel on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing the video URL
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.ServerEnv.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "video uploaded successfully", map[string]any{"id": id, "fileUrl": videoUrl})
}
