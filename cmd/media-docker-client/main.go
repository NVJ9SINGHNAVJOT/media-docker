package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/shutdown"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load env file
	err := pkg.LoadEnv(".env.client")
	if err != nil {
		fmt.Println("Error loading env file", err)
		panic(err)
	}

	// Validate environment variables necessary for the application
	err = config.ValidateClientEnv()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err) // Terminate if environment variables are invalid
	}

	// Set up the logger based on the current environment (development/production)
	config.SetUpLogger(config.ClientEnv.ENVIRONMENT)

	// Check if the media storage directory exists
	pkg.DirExist(helper.Constants.MediaStorage)

	// Initialize a new Chi router for handling HTTP requests
	router := chi.NewRouter()

	// NOTE: Adjust throttle middleware value based on the required traffic control
	// Set up default middlewares such as CORS and logging for the router
	mw.DefaultMiddlewares(router, config.ClientEnv.ALLOWED_ORIGINS, []string{"GET"}, 10000)

	// NOTE: Adjust LimitByIP middleware value based on the allowed requests per IP
	// Apply rate limiting middleware to limit requests by IP (500 requests per minute)
	router.Use(httprate.LimitByIP(500, 1*time.Minute))

	/*
		Create a route along "/media_docker_files" that serves files from the
		./media_docker_files folder. The folder path is constructed based on the
		current working directory and the constant for media storage.
	*/
	workDir, _ := os.Getwd()                                                    // Get the current working directory
	filesDir := http.Dir(filepath.Join(workDir, helper.Constants.MediaStorage)) // Create an HTTP file system directory
	mw.FileServer(router, "/"+helper.Constants.MediaStorage, filesDir)          // Register the file server with the router

	// Define the index handler that responds with a simple message indicating the server is running
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	// Initialize the server port based on the client configuration
	port := ":" + config.ClientEnv.CLIENT_PORT

	// Create a new HTTP server
	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	// Call graceful shutdown function with a goroutine to avoid blocking the main thread
	go shutdown.WaitForShutdownSignal(server, 15)

	// Start the HTTP server
	log.Info().Msg("media-docker-client is running...")
	// Start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("HTTP server crashed.")
	} else {
		log.Info().Msg("Server gracefully stopped.")
	}

	log.Info().Msg("media-docker-client stopped.")
}
