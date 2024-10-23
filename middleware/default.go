package middleware

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// DefaultMiddlewares configures the common middlewares for the router,
// including CORS settings, request logging, panic recovery, and request throttling.
//
// Parameters:
// - router: The chi.Mux router to apply the middlewares.
// - allowedOrigins: A list of origins allowed for CORS requests.
// - allowedMethods: A list of HTTP methods allowed for CORS requests.
// - throttle: Maximum number of requests allowed in parallel to prevent server overload.
func DefaultMiddlewares(router *chi.Mux, allowedOrigins []string, allowedMethods []string, throttle int) {
	// Set up CORS (Cross-Origin Resource Sharing) with the specified allowed origins and methods.
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins, // List of allowed origins for CORS.
		AllowedMethods: allowedMethods, // List of allowed HTTP methods.
		AllowedHeaders: []string{
			"Origin",           // Allow the Origin header.
			"X-Requested-With", // Allow requests with X-Requested-With header.
			"Authorization",    // Allow the Authorization header.
			"Content-Type",     // Allow the Content-Type header.
			"Accept",           // Allow the Accept header.
		},
		AllowCredentials: true, // Allow sending cookies with cross-origin requests.
	}))

	// Add a middleware that generates a unique request ID for each HTTP request.
	router.Use(middleware.RequestID)

	// Add a middleware that logs each request with details such as method, path, and response time.
	router.Use(middleware.Logger)

	// Add a middleware that recovers from panics and prevents the server from crashing.
	router.Use(middleware.Recoverer)

	// Add a middleware to throttle the number of concurrent requests, limiting server overload.
	router.Use(middleware.Throttle(throttle))
}
