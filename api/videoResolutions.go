package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

type videoResolutionsRequest struct {
	ID string `json:"id" validate:"required,uuid4"`
}

// VideoResolutions queues a video for multi-resolution HLS conversion and
// returns its public URLs.
//
// Every URL works immediately, resolving to the raw upload until the ladder is
// promoted into place. Afterwards the base URL serves a master playlist, so a
// player can switch quality on its own, and the per-resolution URLs serve their
// individual playlists.
func VideoResolutions(w http.ResponseWriter, r *http.Request) {
	var req videoResolutionsRequest
	// Parse the JSON request and populate the videoResolutionsRequest struct
	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	meta, ok := claimForDispatch(w, r, asset.TypeVideo, "video-resolutions", req.ID)
	if !ok {
		return
	}

	// Create the VideoResolutionsMessage struct to be passed to Kafka
	message := topics.VideoResolutionsMessage{
		FilePath: originalPathFor(meta), // The raw upload, already publicly served
		NewId:    meta.ID,               // Asset id, minted at upload time
	}

	// Pass the struct to the Kafka producer
	if err := kafkahandler.KafkaProducer.Produce("video-resolutions", message); err != nil {
		dispatchFailed(w, r, asset.TypeVideo, meta.ID, err)
		return
	}

	// Build one URL per rung of the ladder, alongside the adaptive master URL.
	fileUrls := make(map[string]string, len(asset.Resolutions))
	for _, resolution := range asset.Resolutions {
		fileUrls[resolution] = variantURL(asset.TypeVideo, meta.ID, resolution)
	}

	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, "video uploaded successfully",
		map[string]any{
			"id":          meta.ID,
			"fileUrl":     fileURL(asset.TypeVideo, meta.ID), // Master playlist once converted
			"fileUrls":    fileUrls,
			"originalUrl": originalURL(asset.TypeVideo, meta.ID), // Always the upload as it was sent
		})
}
