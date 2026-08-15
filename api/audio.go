package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

type audioRequest struct {
	ID      string  `json:"id" validate:"required,uuid4"`
	Bitrate *string `json:"bitrate" validate:"omitempty,oneof=128k 192k 256k 320k"` // Optional quality parameter
}

// Audio queues an audio file for conversion and returns its public URL.
//
// The URL works immediately, serving the uploaded audio until the converted MP3
// replaces it.
func Audio(w http.ResponseWriter, r *http.Request) {
	var req audioRequest
	// Parse the JSON request and populate the audioRequest struct
	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	meta, ok := claimForDispatch(w, r, asset.TypeAudio, "audio", req.ID)
	if !ok {
		return
	}

	// Create the AudioMessage struct
	message := topics.AudioMessage{
		FilePath: originalPathFor(meta), // The raw upload, already publicly served
		NewId:    meta.ID,               // Asset id, minted at upload time
		Bitrate:  req.Bitrate,           // Optional bitrate (can be nil)
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("audio", message); err != nil {
		dispatchFailed(w, r, asset.TypeAudio, meta.ID, err)
		return
	}

	// Respond with success, providing the stable audio URL
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, "audio uploaded and processed successfully",
		map[string]any{"id": meta.ID, "fileUrl": fileURL(asset.TypeAudio, meta.ID)})
}
