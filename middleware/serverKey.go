package middleware

import (
	"net/http"
	"strings"

	"github.com/nvj9singhnavjot/media-docker/helper"
)

func ServerKey(serverKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			checkKey := r.Header.Get("Authorization")
			checkKey = strings.TrimPrefix(checkKey, "Bearer ")

			if checkKey == serverKey {
				next.ServeHTTP(w, r)
			} else {
				helper.Response(w, http.StatusForbidden, "unauthorized access denied for server", nil)
			}
		})
	}
}
