package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg"
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

	// Set up default middlewares such as CORS and logging for the router
	mw.DefaultMiddlewares(router, config.ClientEnv.ALLOWED_ORIGINS, []string{"GET"}, 5000)

	// Apply rate limiting middleware to limit requests by IP (100 requests per minute)
	router.Use(httprate.LimitByIP(100, 1*time.Minute))

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
	log.Info().Msg("media-docker-client server running...") // Log that the server is starting

	// Create a new HTTP server
	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	go func() {
		// Channel to listen for OS interrupt signals for graceful shutdown
		srvDone := make(chan os.Signal, 1)
		signal.Notify(srvDone, os.Interrupt)

		// Wait for interrupt signal to gracefully shutdown the server
		<-srvDone // Block until a signal is received

		log.Info().Msg("shutting down server...") // Log shutdown process

		// Create a context with a timeout for the shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown the server gracefully
		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("error while shutting down server") // Log error if the shutdown fails
		}
	}()

	// Start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Str("error", err.Error()).Msg("error while running server") // Log fatal error if the server fails to start
	}

	log.Info().Msg("server gracefully stopped") // Log that the server has stopped gracefully
}
