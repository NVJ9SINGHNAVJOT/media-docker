package api

import (
	"fmt"
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/rs/zerolog/log"
)

// fileURL builds the public, stable URL of an asset.
func fileURL(mediaType, id string) string {
	return config.ServerEnv.BASE_URL + asset.URLPath(mediaType, id)
}

// variantURL builds the public URL of one rung of a video's resolution ladder.
func variantURL(mediaType, id, variant string) string {
	return config.ServerEnv.BASE_URL + asset.VariantURLPath(mediaType, id, variant)
}

// originalURL builds the public URL that always serves the raw upload.
//
// Unlike fileURL it never upgrades to converted output, so a caller can offer
// the source bytes alongside the processed ones. It stays valid for as long as
// the asset exists, since conversion never removes the raw upload.
func originalURL(mediaType, id string) string {
	return config.ServerEnv.BASE_URL + asset.OriginalURLPath(mediaType, id)
}

// claimForDispatch checks that an uploaded asset is ready to be handed off for
// processing and marks it as dispatched.
//
// Marking happens before the job is produced, not after. The asset is already
// live in media storage at this point, and the janitor reaps anything still
// flagged undispatched once it ages out; if the flag were only set after a
// successful produce, a slow or failed produce could leave a job queued against
// an asset the janitor then deleted mid-conversion. Getting it wrong in the
// other direction is harmless, so on a produce failure the caller reverts the
// flag via releaseDispatch.
//
// It writes an error response and returns ok=false when the asset is missing or
// has already been dispatched.
func claimForDispatch(w http.ResponseWriter, r *http.Request, mediaType, job, id string) (asset.Meta, bool) {
	meta, err := asset.ReadMeta(mediaType, id)
	if err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "file doesn't exist", err)
		return meta, false
	}

	if meta.Dispatched {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusConflict, "file has already been dispatched", nil)
		return meta, false
	}

	// The raw upload must actually be on disk; metadata alone is not enough.
	exist, err := pkg.DirOrFileExist(asset.OriginalPath(mediaType, id, meta.Ext))
	if err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error checking uploaded file", err)
		return meta, false
	}
	if !exist {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "file doesn't exist", nil)
		return meta, false
	}

	updated, err := asset.MarkDispatched(mediaType, id, job)
	if err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error updating file metadata", err)
		return meta, false
	}

	return updated, true
}

// releaseDispatch reverts the dispatched flag after a job could not be queued,
// so the caller can retry and the janitor can still reap the asset if they never do.
func releaseDispatch(mediaType, id string) {
	meta, err := asset.ReadMeta(mediaType, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Str("type", mediaType).Msg("Error reading meta while releasing dispatch")
		return
	}

	meta.Dispatched = false
	meta.Job = ""

	if err = asset.WriteMeta(meta); err != nil {
		// The asset stays flagged as dispatched and will never be reaped.
		// It remains servable, so this leaks storage rather than breaking a URL.
		log.Error().Err(err).Str("id", id).Str("type", mediaType).Msg("Error releasing dispatch flag")
	}
}

// originalPathFor returns the path of the raw upload a job should read from.
func originalPathFor(meta asset.Meta) string {
	return asset.OriginalPath(meta.Type, meta.ID, meta.Ext)
}

// dispatchFailed writes the response for a job that could not be queued.
func dispatchFailed(w http.ResponseWriter, r *http.Request, mediaType, id string, err error) {
	releaseDispatch(mediaType, id)
	helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError,
		fmt.Sprintf("error sending Kafka message for %s", mediaType), err)
}
