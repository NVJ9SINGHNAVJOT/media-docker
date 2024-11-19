package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
)

type imageRequest struct {
	UuidFilename string `json:"uuidFilename" validate:"required,uuid4"`
}

// Image handles image file upload requests and sends processing messages to Kafka.
func Image(w http.ResponseWriter, r *http.Request) {
	var req imageRequest

	// Parse the JSON request and populate the ImageRequest struct
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid data", err)
		return
	}
	path := helper.Constants.UploadStorage + "/" + req.UuidFilename

	// Check if the file exists at the specified path
	exist, err := pkg.DirOrFileExist(path)
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid uuidFilename", err)
		return
	}

	if !exist {
		helper.Response(w, http.StatusBadRequest, "file doesn't exist", nil)
		return
	}

	id := uuid.New().String()                                                         // Generate a new UUID for the image file
	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, id) // Define the output path for the image file

	// Create the ImageMessage struct to be passed to Kafka
	message := topics.ImageMessage{
		FilePath: path, // Set the file path
		NewId:    id,   // Set the new ID for the file URL
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("image", message); err != nil {
		pkg.AddToFileDeleteChan(path) // Add to deletion channel on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing the image URL
	imageUrl := fmt.Sprintf("%s/%s", config.ServerEnv.BASE_URL, outputPath) // Construct the image file URL
	helper.Response(w, http.StatusCreated, "image uploaded successfully", map[string]any{"id": id, "fileUrl": imageUrl})
}
