package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

type imageRequest struct {
	ID string `json:"id" validate:"required,uuid4"`
	// Compression is the ffmpeg -q:v value, 1 (highest quality) to 31 (lowest).
	Compression *int `json:"compression" validate:"omitempty,min=1,max=31"`
}

// Image queues an image for compression and returns its public URL.
//
// The URL works immediately, serving the uploaded image until the compressed
// JPEG replaces it.
func Image(w http.ResponseWriter, r *http.Request) {
	var req imageRequest

	// Parse the JSON request and populate the imageRequest struct
	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	meta, ok := claimForDispatch(w, r, asset.TypeImage, "image", req.ID)
	if !ok {
		return
	}

	// Create the ImageMessage struct to be passed to Kafka
	message := topics.ImageMessage{
		FilePath:    originalPathFor(meta), // The raw upload, already publicly served
		NewId:       meta.ID,               // Asset id, minted at upload time
		Compression: req.Compression,       // Optional compression level (can be nil)
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("image", message); err != nil {
		dispatchFailed(w, r, asset.TypeImage, meta.ID, err)
		return
	}

	// Respond with success, providing the stable image URL
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, "image uploaded successfully",
		map[string]any{"id": meta.ID, "fileUrl": fileURL(asset.TypeImage, meta.ID)})
}
