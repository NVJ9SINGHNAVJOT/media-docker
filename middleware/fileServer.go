package middleware

import (
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/rs/zerolog/log"
)

// Query parameters the resolver uses to ask for a download and to supply the
// name to download it under.
//
// These only affect the file name shown to the user. Whether a response is an
// attachment at all is decided from the request path, so a client cannot obtain
// an inline response by omitting them.
const (
	DownloadParam     = "download"
	DownloadNameParam = "name"
)

func init() {
	// Go's builtin MIME table has no entry for HLS, and the Alpine-based runtime
	// image has no /etc/mime.types to fall back on, so playlists would otherwise
	// be sniffed as text/plain. hls.js tolerates that; Safari's native player
	// does not always.
	registerMIME(".m3u8", "application/vnd.apple.mpegurl")
	registerMIME(".ts", "video/mp2t")
}

func registerMIME(ext, typ string) {
	if err := mime.AddExtensionType(ext, typ); err != nil {
		log.Warn().Err(err).Str("ext", ext).Msg("Could not register MIME type")
	}
}

// alwaysAttachment lists the URL path segments whose contents are never safe to
// render inline.
//
// The "other" category accepts arbitrary uploads by design, so an uploaded HTML
// or SVG file served inline would execute on the client's origin: stored XSS
// against every site sharing it. Documents get the same treatment for
// consistency and because several document types embed scripts.
var alwaysAttachment = []string{
	"/" + asset.TypeDir(asset.TypeDocument) + "/",
	"/" + asset.TypeDir(asset.TypeOther) + "/",
}

// FileServer sets up a `http.FileServer` handler to serve static files from a given `http.FileSystem`.
// It integrates with the Chi router and configures routes to serve files efficiently.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	// Check if the provided path contains URL parameters (e.g., `{}` or `*`)
	// which are not allowed for static file serving.
	if strings.ContainsAny(path, "{}*") {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			// Respond with a 400 Bad Request status if the path contains URL parameters.
			helper.ErrorResponse(w, helper.GetRequestID(r), 400, "fileServer does not permit any URL parameters.", nil)
		})
		return
	}

	// Ensure that the path ends with a trailing slash for proper routing.
	// If the path does not end with a slash and is not the root path, redirect to the path with a trailing slash.
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		// Update path to include trailing slash.
		path += "/"
	}

	// Append wildcard to the path to match all files under the directory.
	path += "*"

	// Configure the router to handle requests to the specified path.
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		// Never let a browser second-guess the declared content type. Combined
		// with the attachment rule below this is what keeps user-supplied files
		// from executing on this origin.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Decide from the path, not from the query, so the header cannot be
		// avoided by requesting the file directly.
		if isAttachmentPath(r.URL.Path) {
			w.Header().Set("Content-Disposition", contentDisposition(r))
		}

		// Extract the route context to get the route pattern used.
		rctx := chi.RouteContext(r.Context())
		// Remove the trailing wildcard from the route pattern to get the path prefix.
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		// Create a file server handler with the correct prefix for serving files.
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		// Serve the requested file.
		fs.ServeHTTP(w, r)
	})
}

// isAttachmentPath reports whether a request path falls in a category that must
// always be downloaded rather than rendered.
func isAttachmentPath(urlPath string) bool {
	for _, segment := range alwaysAttachment {
		if strings.Contains(urlPath, segment) {
			return true
		}
	}
	return false
}

// contentDisposition builds an attachment header, using the requested download
// name when one was supplied and is safe to embed.
func contentDisposition(r *http.Request) string {
	name := r.URL.Query().Get(DownloadNameParam)
	if name == "" || strings.ContainsAny(name, "\"\\;\r\n") {
		return "attachment"
	}

	return `attachment; filename="` + name + `"`
}
