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

var topics = []string{"videoResponse", "videoResolutionResponse", "imageResponse", "audioResponse"}

func main() {
	// Load env file
	err := pkg.LoadEnv(".env.server")
	if err != nil {
		fmt.Println("Error loading env file", err)
		panic(err)
	}

	// environment variables are checked
	err = config.ValidateServerEnv()
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

	// Initialize WorkerTracker to track remaining workers per topic
	workerTracker := kafka.NewWorkerTracker(config.ServerEnv.KAFKA_GROUP_WORKERS, topics)

	serverKafka.KafkaProducer = kafka.NewKafkaProducerManager(config.ServerEnv.KAFKA_BROKERS)
	serverKafka.KafkaConsumer = kafka.NewKafkaConsumerManager(ctx, errChan, config.ServerEnv.KAFKA_GROUP_WORKERS,
		config.ServerEnv.KAFKA_GROUP_PREFIX_ID, &wg,
		topics, config.ServerEnv.KAFKA_BROKERS, serverKafka.ProcessMessage)

	go pkg.DeleteFileWorker()
	go serverKafka.KafkaConsumer.KafkaConsumeSetup()

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
	router.Use(httprate.LimitByIP(50, 1*time.Minute))

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

	// sync.Once to ensure shutdown happens only once
	var shutdownOnce sync.Once

	// Shutdown handling using signal and worker tracking
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case sig := <-sigChan:
				log.Info().Msgf("Received signal: %s. Shutting down...", sig)

				// Stop accepting new connections immediately
				srv.SetKeepAlivesEnabled(false)

				time.Sleep(1 * time.Minute)
				cancel() // Cancel context to signal Kafka workers to shut down

				// Wait for all Kafka workers to finish before shutting down the server
				log.Info().Msg("Waiting for Kafka workers to complete...")
				wg.Wait() // Wait for all worker goroutines to complete
				log.Info().Msg("All Kafka workers stopped")

				// Consume all remaining error messages from errChan before shutting down
			ConsumeErrors:
				for {
					select {
					case workerErr, ok := <-errChan:
						if ok {
							log.Error().Err(workerErr.Err).Msgf("Kafka worker error for topic: %s, workerName: %s", workerErr.Topic, workerErr.WorkerName)
						} else {
							log.Info().Msg("All remaining Kafka worker errors consumed. Proceeding with shutdown.")
							break ConsumeErrors // Break out of the labeled loop
						}
					default:
						// No more error messages to consume
						log.Info().Msg("No more worker errors to process.")
						break ConsumeErrors // Break out of the labeled loop
					}
				}

				// Now gracefully shut down the HTTP server
				shutdownOnce.Do(func() {
					shutdownServer(srv)
				})
				return

			case workerErr, ok := <-errChan:
				if !ok {
					// If the channel is closed, all workers are done, so shut down
					log.Info().Msg("Worker Error channel closed, All Kafka workers finished. Initiating server shutdown...")
					shutdownOnce.Do(func() {
						shutdownServer(srv)
					})
					return
				}

				log.Error().Err(workerErr.Err).Msgf("Kafka worker error for topic: %s, workerName: %s", workerErr.Topic, workerErr.WorkerName)

				// Reduce worker count for the topic
				remainingWorkers := workerTracker.DecrementWorker(workerErr.Topic)

				if remainingWorkers == 1 {
					log.Warn().Msgf("Only one worker remaining for topic: %s", workerErr.Topic)
				} else if remainingWorkers == 0 {
					log.Error().Msgf("No workers remaining for topic: %s", workerErr.Topic)
				}
			}
		}
	}()

	// Start the HTTP server
	log.Info().Msg("media-docker-server server is running...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("HTTP server crashed")
		cancel() // Ensure context is cancelled if server crashes
	}

	log.Info().Msg("Server stopped")
}

// shutdownServer gracefully shuts down the HTTP server
func shutdownServer(srv *http.Server) {
	pkg.CloseDeleteFileChannel()
	log.Info().Msg("Shutting down the server...")
	// Gracefully shut down the HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}
	log.Info().Msg("Server shut down complete.")
}
