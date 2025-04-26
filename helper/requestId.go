package helper

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// GetRequestID returns the requestId from the context, or "unknown" if not present.
func GetRequestID(r *http.Request) string {
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		return reqID
	}
	return "unknown"
}
