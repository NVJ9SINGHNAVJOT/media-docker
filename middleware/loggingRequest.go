package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// LoggingRequest logs incoming HTTP requests, including method, URL, client IP, headers, and body.
func LoggingRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())

		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			clientIP = r.RemoteAddr // fallback to full RemoteAddr
		}

		requestLog := map[string]interface{}{
			"requestId": reqID,
			"method":    r.Method,
			"url":       r.URL.Path,
			"clientIP":  clientIP,
			"query":     r.URL.Query(),
			"requestHeaders": map[string]string{
				"content-type":       r.Header.Get("content-type"),
				"sec-ch-ua-platform": r.Header.Get("sec-ch-ua-platform"),
				"origin":             strings.TrimSpace(r.Header.Get("origin")),
				"sec-fetch-site":     r.Header.Get("sec-fetch-site"),
				"sec-fetch-mode":     r.Header.Get("sec-fetch-mode"),
			},
		}

		// Only log body if the method is NOT GET
		if r.Method != http.MethodGet && r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				log.Error().
					Err(err).
					Fields(requestLog).
					Msg("Failed to read request body")
				next.ServeHTTP(w, r)
				return
			}

			// Reset body immediately
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			contentType := r.Header.Get("Content-Type")

			switch {
			case strings.HasPrefix(contentType, "application/json"):
				var jsonBody map[string]any
				if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
					requestLog["requestBody"] = jsonBody
				} else {
					requestLog["requestBodyRaw"] = string(bodyBytes)
				}

			case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
				if err := r.ParseForm(); err == nil {
					formData := make(map[string]string)
					for key, values := range r.PostForm {
						formData[key] = strings.Join(values, ", ")
					}
					requestLog["requestBodyForm"] = formData
				} else {
					requestLog["requestBodyRaw"] = string(bodyBytes)
				}

			default:
				requestLog["requestBodyUnknown"] = string(bodyBytes)
			}
		}

		log.Info().
			Fields(requestLog).
			Msg("Incoming Request")

		next.ServeHTTP(w, r)
	})
}
