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

	pkg.DirExist(helper.Constants.UploadStorage)
	pkg.DirExist(helper.Constants.MediaStorage)

	err = kafka.CheckAllKafkaConnections(config.ServerEnv.KAFKA_BROKERS)
	if err != nil {
		log.Fatal().Err(err).Msg("Kafka connection failed for media-docker-server")
	}

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled on shutdown

	// Create a WaitGroup to track worker goroutines
	var wg sync.WaitGroup
	// workDone channel waits for all workers to complete.
	workDone := make(chan int, 1)

	serverKafka.KafkaProducer = kafka.NewKafkaProducerManager(config.ServerEnv.KAFKA_BROKERS)
	serverKafka.KafkaConsumer = kafka.NewKafkaConsumerManager(
		ctx, workDone, config.ServerEnv.KAFKA_TOPIC_WORKERS,
		config.ServerEnv.KAFKA_GROUP_PREFIX_ID, &wg,
		config.ServerEnv.KAFKA_BROKERS, serverKafka.ProcessMessage)

	go pkg.DeleteFileWorker()
	go serverKafka.KafkaConsumer.KafkaConsumeSetup()

	// initialize validator
	helper.InitializeValidator()

	// router intialized
	router := chi.NewRouter()

	// NOTE: Adjust throttle middleware value based on the required traffic control
	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.ServerEnv.ALLOWED_ORIGINS, []string{"POST", "DELETE"}, 5000)

	// server key for accessing server
	router.Use(mw.ServerKey(config.ServerEnv.SERVER_KEY))

	// middlewares for this router
	router.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	router.Use(middleware.AllowContentType("application/json", "multipart/form-data"))
	// NOTE: Adjust LimitByIP middleware value based on the allowed requests per IP
	// Apply rate limiting middleware to limit requests by IP (50 requests per minute)
	router.Use(httprate.LimitByIP(50, 1*time.Minute))

	// all routes for server
	router.Route("/api/v1/uploads", routes.UploadRoutes())
	router.Route("/api/v1/destroys", routes.DestroyRoutes())
	router.Route("/api/v1/connections", routes.ConnectionRoutes())

	// index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	// Setup the server with graceful shutdown
	srv := &http.Server{
		Addr:    ":" + config.ServerEnv.SERVER_PORT,
		Handler: router,
	}

	// sync.Once to ensure cleanUp happens only once
	var cleanUpOnce sync.Once

	// Handle shutdown signals and worker tracking
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case sig := <-sigChan:
				log.Info().Msgf("Received signal: %s. Shutting down...", sig)

				// Gracefully shut down the server
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer shutdownCancel()

				if err := srv.Shutdown(shutdownCtx); err != nil {
					log.Error().Err(err).Msg("HTTP server shutdown error")
				} else {
					log.Info().Msg("Server shut down complete.")
				}

				cancel() // Cancel context to stop Kafka workers
				log.Info().Msg("Waiting for Kafka workers to finish...")

				wg.Wait() // Wait for worker goroutines to complete
				log.Info().Msg("All Kafka workers stopped.")

				// Perform final cleanup
				cleanUpOnce.Do(cleanUpForServer)
				return

			case _, ok := <-workDone:
				if !ok {
					// If the channel is closed, all workers are done, so shut down
					log.Info().Msg("workDone channel closed, all Kafka workers finished. Initiating server shutdown...")

					// Gracefully shut down the server
					shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer shutdownCancel()
					if err := srv.Shutdown(shutdownCtx); err != nil {
						log.Error().Err(err).Msg("HTTP server shutdown error")
					} else {
						log.Info().Msg("Server shut down complete.")
					}
					cleanUpOnce.Do(cleanUpForServer)
					return
				}

			}
		}
	}()

	// Start the HTTP server
	log.Info().Msg("media-docker-server is running...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("HTTP server crashed.")
		cancel()
	}

	log.Info().Msg("Server stopped.")
}

// cleanUpForServer performs any final cleanup actions before shutdown
func cleanUpForServer() {
	// TODO: Notify Kafka consumers to stop
	if err := serverKafka.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("Error while closing producer for media-docker-server")
	} else {
		log.Info().Msg("Producer closed for media-docker-server.")
	}

	pkg.CloseDeleteFileChannel()
	log.Info().Msg("DeleteFile channel closed.")
	time.Sleep(10 * time.Second)
}
