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

// videoRequest represents the structure of the request for video upload.
type videoRequest struct {
	UuidFilename string `json:"uuidFilename" validate:"required,uuid4"`
	Quality      *int   `json:"quality" validate:"omitempty,min=40,max=100"` // Quality must be >= 40 and <= 100
}

// Video handles video upload requests and sends processing messages to Kafka.
func Video(w http.ResponseWriter, r *http.Request) {

	var req videoRequest
	// Parse the JSON request and populate the VideoRequest struct
	if err := helper.ValidateRequest(r, &req); err != nil {
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

	id := uuid.New().String()                                                    // Generate a new UUID for the video
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, id) // Define the output path for the video

	// Create the VideoMessage struct to be passed to Kafka
	message := topics.VideoMessage{
		FilePath: path,        // Set the file path
		NewId:    id,          // Set the new ID
		Quality:  req.Quality, // Set the optional quality (can be nil)
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("video", message); err != nil {
		pkg.AddToFileDeleteChan(path) // Add to deletion channel on error
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing the video URL
	videoUrl := fmt.Sprintf("%s/%s/index.m3u8", config.ServerEnv.BASE_URL, outputPath) // Construct the video file URL
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, "video uploaded successfully", map[string]any{"id": id, "fileUrl": videoUrl})
}
