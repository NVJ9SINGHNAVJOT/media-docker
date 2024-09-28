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

// ImageMessage represents the structure of the message sent to Kafka for image processing.
type ImageMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New unique identifier for the image file URL
}

// Image handles image file upload requests and sends processing messages to Kafka.
func Image(w http.ResponseWriter, r *http.Request) {
	// Read the image file from the request
	_, header, err := r.FormFile("imageFile")
	if err != nil {
		// Respond with an error if the file cannot be read
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}
	imagePath := header.Header.Get("path") // Get the file path from the header

	id := uuid.New().String()                                                         // Generate a new UUID for the image file
	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, id) // Define the output path for the image file

	// Create the ImageMessage struct to be passed to Kafka
	message := ImageMessage{
		FilePath: imagePath, // Set the file path
		NewId:    id,        // Set the new ID for the file URL
	}

	// Create a channel of size 1 to store the Kafka response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the id as the key
	serverKafka.ImageRequestMap.Store(id, responseChannel)

	// Pass the struct to the Kafka producer
	if err := serverKafka.KafkaProducer.Produce("image", message); err != nil {
		pkg.AddToFileDeleteChan(imagePath)     // Add to deletion channel on error
		serverKafka.ImageRequestMap.Delete(id) // Remove the channel from the map on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err.Error())
		return
	}

	// Wait for the response from the Kafka processor
	responseSuccess := <-responseChannel

	// Delete the channel from the map once processing is complete
	serverKafka.ImageRequestMap.Delete(id)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "image conversion failed", nil)
		return
	}

	// Respond with success, providing the image URL
	imageUrl := fmt.Sprintf("%s/%s", config.ServerEnv.BASE_URL, outputPath) // Construct the image file URL
	helper.Response(w, http.StatusCreated, "image uploaded successfully", map[string]any{"fileUrl": imageUrl})
}
