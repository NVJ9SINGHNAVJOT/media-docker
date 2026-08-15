package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

// storeRequest is the request shape for the categories that are stored as-is.
type storeRequest struct {
	ID string `json:"id" validate:"required,uuid4"`
}

// Document finalises a document upload.
//
// Documents are never converted, so there is no job to queue and no consumer to
// wait for: the file is already in its final location by the time this is
// called. All this endpoint does is mark the upload as claimed, so the janitor
// stops treating it as abandoned, and hand back the URL it is already served
// from.
func Document(w http.ResponseWriter, r *http.Request) {
	storeAsIs(w, r, asset.TypeDocument, "document")
}

// Other finalises an upload of an arbitrary file type. See Document; the only
// difference is the storage category and the absence of a MIME allowlist.
func Other(w http.ResponseWriter, r *http.Request) {
	storeAsIs(w, r, asset.TypeOther, "other")
}

// storeAsIs completes an upload for a category that needs no processing.
func storeAsIs(w http.ResponseWriter, r *http.Request, mediaType, fileType string) {
	var req storeRequest

	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	meta, ok := claimForDispatch(w, r, mediaType, fileType, req.ID)
	if !ok {
		return
	}

	// Nothing will ever change this asset's representation, so the URL returned
	// here is already final rather than an upgrade waiting to happen.
	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusCreated, fileType+" uploaded successfully",
		map[string]any{"id": meta.ID, "fileUrl": fileURL(mediaType, meta.ID)})
}
