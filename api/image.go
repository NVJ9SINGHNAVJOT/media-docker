package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/kafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

type imageMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New ID for image URL
}

func Image(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("imageFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}
	imagePath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, id)

	// Create the imageMessage struct to be passed to Kafka
	message := imageMessage{
		FilePath: imagePath, // Set the file path
		NewId:    id,
	}

	// Create a channel of size 1 to store the Kafka response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the file path as the key
	kafka.RequestMap.Store(imagePath, responseChannel)

	// Pass the struct to the Kafka producer
	if err := kafka.ProduceKafkaMessage("image", message); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err.Error())
		kafka.RequestMap.Delete(imagePath)
		go pkg.DeleteFile(imagePath)
		return
	}

	// Wait for the response from the Kafka processor
	responseSuccess := <-responseChannel

	// Delete the channel from the map once processing is complete
	kafka.RequestMap.Delete(imagePath)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "image conversion failed", nil)
		go pkg.DeleteFile(imagePath)
		return
	}

	// Respond with success
	imageUrl := fmt.Sprintf("%s/%s", config.MDSenvs.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "image uploaded successfully", map[string]any{"fileUrl": imageUrl})

	go pkg.DeleteFile(imagePath)
}
