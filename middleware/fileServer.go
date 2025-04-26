package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/helper"
)

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
