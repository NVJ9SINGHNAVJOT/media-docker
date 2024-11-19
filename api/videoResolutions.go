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

type videoResolutionsRequest struct {
	UuidFilename string `json:"uuidFilename" validate:"required,uuid4"`
}

// VideoResolutions handles video file upload requests and sends processing messages to Kafka for resolution conversion.
func VideoResolutions(w http.ResponseWriter, r *http.Request) {
	var req videoResolutionsRequest
	// Parse the JSON request and populate the VideoResolutionsRequest struct
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

	id := uuid.New().String() // Generate a new UUID for the video

	// Create the VideoResolutionsMessage struct to be passed to Kafka
	message := topics.VideoResolutionsMessage{
		FilePath: path, // Set the file path
		NewId:    id,   // Set the new ID for the file URL
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("video-resolutions", message); err != nil {
		pkg.AddToFileDeleteChan(path) // Add to deletion channel on error
		helper.Response(w, http.StatusInternalServerError, "error sending Kafka message", err)
		return
	}

	// Respond with success, providing URLs for different video resolutions
	helper.Response(w, http.StatusCreated, "video uploaded successfully",
		map[string]any{
			"id": id,
			"fileUrls": map[string]string{
				"360":  fmt.Sprintf("%s/%s/videos/%s/360/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
				"480":  fmt.Sprintf("%s/%s/videos/%s/480/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
				"720":  fmt.Sprintf("%s/%s/videos/%s/720/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
				"1080": fmt.Sprintf("%s/%s/videos/%s/1080/index.m3u8", config.ServerEnv.BASE_URL, helper.Constants.MediaStorage, id),
			},
		})
}
