package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/mediadockerkafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
)

// Image handles image file upload requests and sends processing messages to Kafka.
func Image(w http.ResponseWriter, r *http.Request) {
	// Read the image file from the request
	_, header, err := r.FormFile("imageFile")
	if err != nil {
		// Respond with an error if the file cannot be read
		helper.Response(w, http.StatusBadRequest, "error reading file", err)
		return
	}
	imagePath := header.Header.Get("path") // Get the file path from the header

	id := uuid.New().String()                                                         // Generate a new UUID for the image file
	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, id) // Define the output path for the image file

	// Create the ImageMessage struct to be passed to Kafka
	message := topics.ImageMessage{
		FilePath: imagePath, // Set the file path
		NewId:    id,        // Set the new ID for the file URL
	}

	// Pass the struct to the Kafka producer
	if err := mediadockerkafka.KafkaProducer.Produce("image", message); err != nil {
		pkg.AddToFileDeleteChan(imagePath) // Add to deletion channel on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing the image URL
	imageUrl := fmt.Sprintf("%s/%s", config.ServerEnv.BASE_URL, outputPath) // Construct the image file URL
	helper.Response(w, http.StatusCreated, "image uploaded successfully", map[string]any{"id": id, "fileUrl": imageUrl})
}
