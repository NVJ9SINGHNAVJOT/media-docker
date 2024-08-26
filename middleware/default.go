package middleware

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func DefaultMiddlewares(allowedOrigins []string, router *chi.Mux) {
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "DELETE"},
		AllowedHeaders: []string{
			"Origin",
			"X-Requested-With",
			"Authorization",
			"Content-Type",
			"Refresh-Token",
			"Accept"},
		AllowCredentials: true,
	}))
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Throttle(1000))
	router.Use(httprate.LimitByIP(10, 1*time.Minute))
	router.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	router.Use(middleware.AllowContentType("application/json", "multipart/form-data"))
}
