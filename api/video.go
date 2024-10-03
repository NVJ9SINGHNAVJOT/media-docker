package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

// videoRequest represents the structure of the request for video upload.
type videoRequest struct {
	Quality string `json:"quality" validate:"omitempty,customVideoQuality"` // Optional quality parameter
}

// VideoMessage represents the structure of the message sent to Kafka for video processing.
type VideoMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New unique identifier for the video file URL
	Quality  *int   `json:"quality" validate:"omitempty"` // Optional video quality (using pointer for omitempty)
}

// Video handles video upload requests and sends processing messages to Kafka.
func Video(w http.ResponseWriter, r *http.Request) {
	// Read the video file from the request
	_, header, err := r.FormFile("videoFile")
	if err != nil {
		// Respond with an error if the file cannot be read
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}
	videoPath := header.Header.Get("path") // Get the file path from the header

	var req videoRequest
	// Parse the JSON request and populate the VideoRequest struct
	if err := helper.ValidateRequest(r, &req); err != nil {
		pkg.AddToFileDeleteChan(videoPath) // Add to deletion channel on error
		helper.Response(w, http.StatusBadRequest, "invalid data", err.Error())
		return
	}

	// Initialize quality as nil for optional use
	var quality *int = nil
	// Convert quality string to an integer if provided
	if req.Quality != "" {
		q, err := strconv.Atoi(req.Quality) // Convert string to int
		if err != nil {
			pkg.AddToFileDeleteChan(videoPath) // Add to deletion channel on error
			helper.Response(w, http.StatusBadRequest, "invalid quality value", err.Error())
			return
		}
		quality = &q // Set quality as a pointer to the integer value
	}

	id := uuid.New().String()                                                    // Generate a new UUID for the video
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, id) // Define the output path for the video

	// Create the VideoMessage struct to be passed to Kafka
	message := VideoMessage{
		FilePath: videoPath, // Set the file path
		NewId:    id,        // Set the new ID
		Quality:  quality,   // Set the optional quality (can be nil)
	}

	// Create a channel of size 1 to store the Kafka response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the id as the key
	serverKafka.VideoRequestMap.Store(id, responseChannel)

	// Pass the struct to the Kafka producer
	if err := serverKafka.KafkaProducer.Produce("video", message); err != nil {
		pkg.AddToFileDeleteChan(videoPath)     // Add to deletion channel on error
		serverKafka.VideoRequestMap.Delete(id) // Remove the channel from the map on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err.Error())
		return
	}

	// Wait for the response from the Kafka processor
	responseSuccess := <-responseChannel

	// Delete the channel from the map once processing is complete
	serverKafka.VideoRequestMap.Delete(id)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "video conversion failed", nil)
		return
	}

	// Respond with success, providing the video URL
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.ServerEnv.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "video uploaded successfully", map[string]any{"fileUrl": videoUrl})
}
