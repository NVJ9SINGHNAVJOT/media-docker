package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/routes"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	"github.com/nvj9singhnavjot/media-docker/kafka"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

func main() {

	// environment variables are checked
	err := config.ValidateServerEnv()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	// logger setup for server
	config.SetUpLogger(config.ServerEnv.ENVIRONMENT)

	exist, err := pkg.DirExist(helper.Constants.UploadStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("error while checking /" + helper.Constants.UploadStorage + " dir")
	} else if !exist {
		log.Fatal().Msg(helper.Constants.UploadStorage + " dir does not exist")
	}

	exist, err = pkg.DirExist(helper.Constants.MediaStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("error while checking /" + helper.Constants.MediaStorage + " dir")
	} else if !exist {
		log.Fatal().Msg(helper.Constants.MediaStorage + " dir does not exist")
	}

	err = kafka.CheckAllKafkaConnections(config.ServerEnv.KAFKA_BROKERS)
	if err != nil {
		log.Fatal().Err(err).Msg("Kafka connection failed for media-docker-server")
	}

	// Create a WaitGroup to track worker goroutines
	var wg sync.WaitGroup

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled on shutdown

	// Error channel to listen to Kafka worker errors
	errChan := make(chan kafka.WorkerError)

	topics := []string{"videoResponse", "videoResolutionResponse", "imageResponse", "audioResponse"}
	// Initialize WorkerTracker to track remaining workers per topic
	workerTracker := kafka.NewWorkerTracker(config.ServerEnv.KAFKA_GROUP_WORKERS, topics)

	serverKafka.KafkaProducer = kafka.NewKafkaProducerManager(config.ServerEnv.KAFKA_BROKERS)
	serverKafka.KafkaConsumer = kafka.NewKafkaConsumerManager(ctx, errChan, config.ServerEnv.KAFKA_GROUP_WORKERS,
		config.ServerEnv.KAFKA_GROUP_PREFIX_ID, &wg,
		topics, config.ServerEnv.KAFKA_BROKERS, serverKafka.ProcessMessage)

	// initialize validator
	helper.InitializeValidator()

	// router intialized
	router := chi.NewRouter()

	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.ServerEnv.ALLOWED_ORIGINS, []string{"POST", "DELETE"}, 1000)

	// server key for accessing server
	router.Use(mw.ServerKey(config.ServerEnv.SERVER_KEY))

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
		Addr:    ":" + config.ServerEnv.SERVER_PORT,
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

	// Shutdown the HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	} else {
		log.Info().Msg("HTTP server stopped gracefully")
	}
}
