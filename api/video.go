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

type videoRequest struct {
	Quality string `json:"quality" validate:"omitempty,customVideoQuality"`
}

// Struct for Kafka message// Struct for Kafka message with quality as an optional number
type VideoMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New id for file url
	Quality  *int   `json:"quality" validate:"omitempty"` // Optional field for the video quality (using pointer for omitempty)
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

	// Initialize quality as nil for optional use
	var quality *int

	// Convert quality string to an integer if provided
	if req.Quality != "" {
		q, err := strconv.Atoi(req.Quality) // Convert string to int
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "invalid quality value", err.Error())
			go pkg.DeleteFile(videoPath)
			return
		}
		quality = &q // Set quality as a pointer to the integer value
	}

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, id)

	// Create the VideoMessage struct to be passed to Kafka
	message := VideoMessage{
		FilePath: videoPath, // Set the file path
		NewId:    id,
		Quality:  quality, // Set the optional quality (can be nil)
	}

	// Create a channel of size 1 to store the Kafka response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the id as the key
	serverKafka.VideoRequestMap.Store(id, responseChannel)

	// Pass the struct to the Kafka producer
	if err := serverKafka.KafkaProducer.Produce("video", message); err != nil {
		pkg.AddToFileDeleteChan(videoPath)
		serverKafka.VideoRequestMap.Delete(videoPath)
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err.Error())
		return
	}

	// Wait for the response from the Kafka processor
	responseSuccess := <-responseChannel

	// Delete the channel from the map once processing is complete
	serverKafka.VideoRequestMap.Delete(videoPath)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "video conversion failed", nil)
		return
	}

	// Respond with success
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.ServerEnv.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "video uploaded successfully", map[string]any{"fileUrl": videoUrl})
}
