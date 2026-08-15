// Package routes wires the media-docker-client's HTTP routes.
package routes

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/helper"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
)

// ResolveRoutes serves the stable public URL of every asset.
//
// This is what makes a media-docker URL usable the instant an upload finishes.
// Rather than pointing callers at a file that does not exist yet, the URL is
// resolved per request against what is actually on disk: the raw upload while
// conversion is pending, the converted output once a consumer has promoted it.
//
// Resolution is a redirect rather than a proxy so that relative references
// inside an HLS playlist keep working. The redirect lands inside the asset's own
// directory, so "segment000.ts" next to "index.m3u8" resolves correctly; serving
// the playlist body from the resolver URL would break those references.
func ResolveRoutes() func(router chi.Router) {
	return func(router chi.Router) {
		for _, mediaType := range []string{
			asset.TypeVideo, asset.TypeImage, asset.TypeAudio, asset.TypeDocument, asset.TypeOther,
		} {
			dir := asset.TypeDir(mediaType)

			router.Get("/"+dir+"/{id}", resolveHandler(mediaType, false))

			// Videos additionally expose each rung of the resolution ladder.
			if mediaType == asset.TypeVideo {
				router.Get("/"+dir+"/{id}/{variant}", resolveHandler(mediaType, true))
			}
		}
	}
}

// resolveHandler builds the handler for one media type.
func resolveHandler(mediaType string, withVariant bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var variant string
		if withVariant {
			variant = chi.URLParam(r, "variant")
			// Reject anything that is not a known rung rather than silently
			// falling back, so a typo does not quietly serve a different quality.
			if !asset.IsResolution(variant) {
				helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusNotFound, "unknown resolution", nil)
				return
			}
		}

		resolution, err := asset.Resolve(mediaType, id, variant)
		if err != nil {
			if errors.Is(err, asset.ErrNotFound) {
				helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusNotFound, "file not found", nil)
				return
			}
			helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error resolving file", err)
			return
		}

		// An unconverted asset can change representation at any moment, so the
		// redirect must never be cached: a client that cached the raw file would
		// never pick up the converted output. Converted output is terminal and
		// may be cached.
		if resolution.Converted {
			w.Header().Set("Cache-Control", "public, max-age=300")
		} else {
			w.Header().Set("Cache-Control", "no-store")
		}

		// Documents and arbitrary uploads are downloads, never inline content.
		// See downloadQuery for why this matters.
		target := "/" + helper.Constants.MediaStorage + "/" + resolution.RelPath
		if mediaType == asset.TypeDocument || mediaType == asset.TypeOther {
			target += downloadQuery(resolution.Meta.OriginalName)
		}

		http.Redirect(w, r, target, http.StatusFound)
	}
}

// downloadQuery builds the query string that tells the file server to send an
// asset as an attachment under its original name.
func downloadQuery(originalName string) string {
	q := url.Values{}
	q.Set(mw.DownloadParam, "1")

	// Only pass a name that cannot influence the header beyond its own value.
	if name := sanitizeDownloadName(originalName); name != "" {
		q.Set(mw.DownloadNameParam, name)
	}

	return "?" + q.Encode()
}

// sanitizeDownloadName strips anything that could break out of the
// Content-Disposition filename parameter, including path separators, quotes and
// control characters.
func sanitizeDownloadName(name string) string {
	if name == "" {
		return ""
	}

	// Never let a supplied name carry path information.
	name = name[strings.LastIndexAny(name, `/\`)+1:]

	cleaned := strings.Map(func(rn rune) rune {
		if rn < 0x20 || rn == 0x7f || rn == '"' || rn == ';' || rn == '\\' {
			return -1
		}
		return rn
	}, name)

	if len(cleaned) > 100 {
		cleaned = cleaned[:100]
	}

	return cleaned
}
