package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/routes"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/worker"
	"github.com/rs/zerolog/log"
)

func main() {

	// environment variables are checked
	err := config.ValidateMDSenvs()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	// logger setup for server
	config.SetUpLogger(config.MDSenvs.ENVIRONMENT)

	// check dir setup for server
	config.CreateDirSetup()

	// Set channels for command execution.
	// Each channel has its own independent pool of workers.
	// NOTE: Start worker setup in a goroutine
	go worker.SetupChannels() // This will use all available cores

	time.Sleep(5 * time.Second)
	// HACK: Set max cores for the HTTP server
	// Limit the server to 1 core
	runtime.GOMAXPROCS(1)

	// initialize validator
	helper.InitializeValidator()

	// router intialized
	router := chi.NewRouter()

	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.MDSenvs.ALLOWED_ORIGINS_SERVER, []string{"POST", "DELETE"}, 1000)

	// server key for accessing server
	router.Use(mw.ServerKey(config.MDSenvs.SERVER_KEY))

	// middlewares for this router
	router.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	router.Use(middleware.AllowContentType("application/json", "multipart/form-data"))
	router.Use(httprate.LimitByIP(10, 1*time.Minute))

	// all routes for server
	router.Route("/api/v1/uploads", routes.UploadRoutes())
	router.Route("/api/v1/destroys", routes.DestroyRoutes())
	router.Route("/api/v1/connections", routes.ConnectionRoutes())

	// index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	// Setup the server with a graceful shutdown
	srv := &http.Server{
		Addr:    ":" + config.MDSenvs.SERVER_PORT,
		Handler: router,
	}

	go shutdownGracefully(srv)
	log.Info().Msg("media-docker-server server running...")

	// Start the HTTP server
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("HTTP server crashed")
	}

	log.Info().Msg("Server stopped")
}

// shutdownGracefully handles server shutdown upon receiving interrupt signals
func shutdownGracefully(server *http.Server) {
	// Channel to listen for OS signals
	signalChan := make(chan os.Signal, 1)
	// Notify for SIGINT (Ctrl+C) or SIGTERM (termination signal)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal
	<-signalChan
	log.Info().Msg("Received shutdown signal, gracefully shutting down...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info().Msg("Stopping all workers for all channels.")

	// Close all worker channels to stop workers
	worker.CloseChannels()

	// Ensure all worker goroutines finish
	log.Info().Msg("All workers have stopped, exiting application.")

	// Shutdown the HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	} else {
		log.Info().Msg("HTTP server stopped gracefully")
	}
}
