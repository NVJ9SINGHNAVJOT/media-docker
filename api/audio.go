package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

type audioRequest struct {
	Bitrate string `json:"bitrate" validate:"omitempty,oneof=128k 192k 256k 320k"` // Optional quality parameter
}

// AudioMessage represents the structure of the message sent to Kafka for audio processing.
type AudioMessage struct {
	FilePath string  `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string  `json:"newId" validate:"required"`    // New unique identifier for the audio file URL
	Bitrate  *string `json:"bitrate" validate:"omitempty"` // Optional quality parameter
}

// Audio handles audio file upload requests and sends processing messages to Kafka.
func Audio(w http.ResponseWriter, r *http.Request) {
	// Read the audio file from the request
	_, header, err := r.FormFile("audioFile")
	if err != nil {
		// Respond with an error if the file cannot be read
		helper.Response(w, http.StatusBadRequest, "error reading file", err)
		return
	}
	audioPath := header.Header.Get("path") // Get the file path from the header

	var req audioRequest
	// Parse the JSON request and populate the AudioRequest struct
	if err := helper.ValidateRequest(r, &req); err != nil {
		pkg.AddToFileDeleteChan(audioPath) // Add to deletion channel on error
		helper.Response(w, http.StatusBadRequest, "invalid data", err)
		return
	}

	id := uuid.New().String()                                                        // Generate a new UUID for the audio file
	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, id) // Define the output path for the audio file

	// Create the AudioMessage struct without bitrate
	message := AudioMessage{
		FilePath: audioPath, // Set the file path
		NewId:    id,        // Set the new ID for the file URL
	}

	// If a bitrate is provided, set it in the message
	if req.Bitrate != "" {
		message.Bitrate = &req.Bitrate
	}

	// Create a channel of size 1 to store the Kafka-like processing response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the id as the key
	serverKafka.AudioRequestMap.Store(id, responseChannel)

	// Pass the struct to the Kafka producer
	if err := serverKafka.KafkaProducer.Produce("audio", message); err != nil {
		pkg.AddToFileDeleteChan(audioPath)     // Add to deletion channel on error
		serverKafka.AudioRequestMap.Delete(id) // Remove the channel from the map on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Wait for the response from the Kafka processor or worker
	responseSuccess := <-responseChannel

	// Delete the channel from the map once processing is complete
	serverKafka.AudioRequestMap.Delete(id)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "audio conversion failed", nil)
		return
	}

	// Respond with success, providing the audio URL
	audioUrl := fmt.Sprintf("%s/%s", config.ServerEnv.BASE_URL, outputPath) // Construct the audio file URL
	helper.Response(w, http.StatusCreated, "audio uploaded and processed successfully", map[string]any{"fileUrl": audioUrl})
}
