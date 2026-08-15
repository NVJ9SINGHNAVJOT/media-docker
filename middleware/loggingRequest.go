package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/rs/zerolog/log"
)

const (
	// maxLoggedFieldValue caps a single logged form value so a large text field
	// cannot flood the log.
	maxLoggedFieldValue = 256

	// maxLoggedFields caps how many form keys are logged per request.
	maxLoggedFields = 20

	// maxLoggedRawBody caps the unparseable-JSON fallback.
	maxLoggedRawBody = 2048
)

// truncate shortens s to limit characters, marking it when it was cut.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "...(truncated)"
}

// loggedFile is the metadata recorded for an uploaded part. The part itself is
// never opened -- media bytes must not reach the logs.
type loggedFile struct {
	Field    string `json:"field"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

// LoggingRequest logs incoming HTTP requests: method, URL, client IP, headers, and
// a safe summary of the body.
//
// CAUTION: uploads arrive as multipart/form-data carrying raw media. That body is
// never read here -- only the non-file form values and each part's name and size
// are logged. Reading it would both dump binary into the log and buffer the whole
// upload in memory ahead of the handler.
func LoggingRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())

		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			clientIP = r.RemoteAddr // fallback to full RemoteAddr
		}

		contentType := r.Header.Get("Content-Type")

		requestLog := map[string]interface{}{
			"requestId":     reqID,
			"method":        r.Method,
			"url":           r.URL.Path,
			"clientIP":      clientIP,
			"query":         r.URL.Query(),
			"contentLength": r.ContentLength,
			"requestHeaders": map[string]string{
				"content-type":       contentType,
				"sec-ch-ua-platform": r.Header.Get("sec-ch-ua-platform"),
				"origin":             strings.TrimSpace(r.Header.Get("origin")),
				"sec-fetch-site":     r.Header.Get("sec-fetch-site"),
				"sec-fetch-mode":     r.Header.Get("sec-fetch-mode"),
			},
		}

		// Only summarize a body if the method is NOT GET
		if r.Method != http.MethodGet && r.Body != nil {
			switch {
			case strings.HasPrefix(contentType, "multipart/form-data"):
				logMultipart(r, requestLog)

			case strings.HasPrefix(contentType, "application/json"):
				bodyBytes, err := readAndRestoreBody(r)
				if err != nil {
					log.Error().
						Err(err).
						Fields(requestLog).
						Msg("Failed to read request body")
					next.ServeHTTP(w, r)
					return
				}

				var jsonBody map[string]any
				if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
					requestLog["requestBody"] = jsonBody
				} else {
					requestLog["requestBodyRaw"] = truncate(string(bodyBytes), maxLoggedRawBody)
				}

			case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
				bodyBytes, err := readAndRestoreBody(r)
				if err != nil {
					log.Error().
						Err(err).
						Fields(requestLog).
						Msg("Failed to read request body")
					next.ServeHTTP(w, r)
					return
				}

				if err := r.ParseForm(); err == nil {
					formData := make(map[string]string)
					for key, values := range r.PostForm {
						formData[key] = truncate(strings.Join(values, ", "), maxLoggedFieldValue)
					}
					requestLog["requestBodyForm"] = formData
				} else {
					requestLog["requestBodyRaw"] = truncate(string(bodyBytes), maxLoggedRawBody)
				}

			default:
				// Unknown content type: the body may be anything, so it is not read.
				requestLog["requestBodyOmitted"] = contentType
			}
		}

		log.Info().
			Fields(requestLog).
			Msg("Incoming Request")

		next.ServeHTTP(w, r)
	})
}

// readAndRestoreBody reads the request body and replaces it so the handler can
// read it again. Only for bodies known to be small enough to buffer.
func readAndRestoreBody(r *http.Request) ([]byte, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Reset body immediately
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes, nil
}

// logMultipart records the non-file fields and the per-part metadata of a
// multipart request, without touching the uploaded bytes.
//
// ParseMultipartForm is safe to call here: it is a no-op once the form has been
// parsed, so the upload handlers that call it again receive the same parsed form.
func logMultipart(r *http.Request, requestLog map[string]interface{}) {
	if err := r.ParseMultipartForm(helper.Constants.MaxChunkSize); err != nil {
		requestLog["multipartParseError"] = err.Error()
		return
	}

	if r.MultipartForm == nil {
		return
	}

	formFields := make(map[string]string, len(r.MultipartForm.Value))
	for key, values := range r.MultipartForm.Value {
		if len(formFields) >= maxLoggedFields {
			formFields["..."] = "more fields omitted"
			break
		}
		formFields[key] = truncate(strings.Join(values, ", "), maxLoggedFieldValue)
	}
	if len(formFields) > 0 {
		requestLog["formFields"] = formFields
	}

	files := make([]loggedFile, 0, len(r.MultipartForm.File))
	for field, headers := range r.MultipartForm.File {
		for _, header := range headers {
			files = append(files, loggedFile{
				Field:    field,
				FileName: header.Filename,
				Size:     header.Size,
			})
		}
	}
	if len(files) > 0 {
		requestLog["files"] = files
	}
}
