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

// Struct for Kafka message without bitrate field
type audioMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New id for file url
}

func Audio(w http.ResponseWriter, r *http.Request) {

	_, header, err := r.FormFile("audioFile")
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "error reading file", err.Error())
		return
	}
	audioPath := header.Header.Get("path")

	id := uuid.New().String()
	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, id)

	// Create the AudioMessage struct without bitrate
	message := audioMessage{
		FilePath: audioPath, // Set the file path
		NewId:    id,        // Set the new ID for file URL
	}

	// Create a channel of size 1 to store the Kafka-like processing response for this request
	responseChannel := make(chan bool, 1)

	// Store the channel in the request map with the file path as the key
	kafka.RequestMap.Store(audioPath, responseChannel)

	// Pass the struct to the Kafka producer (or processing worker)
	if err := kafka.ProduceKafkaMessage("audio", message); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err.Error())
		kafka.RequestMap.Delete(audioPath)
		go pkg.DeleteFile(audioPath)
		return
	}

	// Wait for the response from the Kafka processor or worker
	responseSuccess := <-responseChannel

	// Delete the channel from the map once processing is complete
	kafka.RequestMap.Delete(audioPath)

	// Check if the processing was successful or failed
	if !responseSuccess {
		helper.Response(w, http.StatusInternalServerError, "audio conversion failed", nil)
		go pkg.DeleteFile(audioPath)
		return
	}

	// Respond with success
	audioUrl := fmt.Sprintf("%s/%s", config.MDSenvs.BASE_URL, outputPath)
	helper.Response(w, http.StatusCreated, "audio uploaded and processed successfully", map[string]any{"fileUrl": audioUrl})

	go pkg.DeleteFile(audioPath)
}
