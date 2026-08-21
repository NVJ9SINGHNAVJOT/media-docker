package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

// videoRequest represents the structure of the request for video processing.
type videoRequest struct {
	ID      string `json:"id" validate:"required,uuid4"`
	Quality *int   `json:"quality" validate:"omitempty,min=40,max=100"` // Quality must be >= 40 and <= 100
}

// Video queues a video for HLS conversion and returns its public URL.
//
// The URL works immediately: the raw upload is already in media storage and is
// served until the consumer promotes the converted playlist into place. Callers
// no longer need to wait for a Kafka response before using it.
func Video(w http.ResponseWriter, r *http.Request) {
	var req videoRequest
	// Parse the JSON request and populate the videoRequest struct
	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	meta, ok := claimForDispatch(w, r, asset.TypeVideo, "video", req.ID)
	if !ok {
		return
	}

	// Create the VideoMessage struct to be passed to Kafka
	message := topics.VideoMessage{
		FilePath: originalPathFor(meta), // The raw upload, already publicly served
		NewId:    meta.ID,               // Asset id, minted at upload time
		Quality:  req.Quality,           // Optional quality (can be nil)
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("video", message); err != nil {
		dispatchFailed(w, r, asset.TypeVideo, meta.ID, err)
		return
	}

	// Respond with success, providing the stable video URL alongside the URL that
	// keeps serving the upload as it was sent.
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, "video uploaded successfully",
		map[string]any{
			"id":          meta.ID,
			"fileUrl":     fileURL(asset.TypeVideo, meta.ID),
			"originalUrl": originalURL(asset.TypeVideo, meta.ID),
		})
}
