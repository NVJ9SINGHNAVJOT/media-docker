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
	"github.com/nvj9singhnavjot/media-docker/validator"
)

type audioRequest struct {
	UuidFilename string  `json:"uuidFilename" validate:"required,uuid4"`
	Bitrate      *string `json:"bitrate" validate:"omitempty,oneof=128k 192k 256k 320k"` // Optional quality parameter
}

// Audio handles audio file upload requests and sends processing messages to Kafka.
func Audio(w http.ResponseWriter, r *http.Request) {
	var req audioRequest
	// Parse the JSON request and populate the AudioRequest struct
	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	path := helper.Constants.UploadStorage + "/" + req.UuidFilename

	// Check if the file exists at the specified path
	exist, err := pkg.DirOrFileExist(path)
	if err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid uuidFilename", err)
		return
	}

	if !exist {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "file doesn't exist", nil)
		return
	}

	id := uuid.New().String()                                                        // Generate a new UUID for the audio file
	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, id) // Define the output path for the audio file

	// Create the AudioMessage struct
	message := topics.AudioMessage{
		FilePath: path,        // Set the file path
		NewId:    id,          // Set the new ID for the file URL
		Bitrate:  req.Bitrate, // Set the bitrate if provided in the request
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("audio", message); err != nil {
		pkg.AddToFileDeleteChan(path) // Add to deletion channel on error
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing the audio URL
	audioUrl := fmt.Sprintf("%s/%s", config.ServerEnv.BASE_URL, outputPath) // Construct the audio file URL
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, "audio uploaded and processed successfully", map[string]any{"id": id, "fileUrl": audioUrl})
}
