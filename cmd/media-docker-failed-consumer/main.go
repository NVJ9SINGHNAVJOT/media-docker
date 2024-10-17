package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-failed-consumer/process"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load env file
	err := pkg.LoadEnv(".env.failed")
	if err != nil {
		fmt.Println("Error loading env file", err)
		panic(err)
	}

	// Validate environment variables
	err = config.ValidateFailedConsumeEnv()
	if err != nil {
		fmt.Println("Invalid environment variables", err)
		panic(err)
	}

	// Setup logger
	config.SetUpLogger(config.FailedConsumeEnv.ENVIRONMENT)

	pkg.DirExist(helper.Constants.UploadStorage)
	pkg.DirExist(helper.Constants.MediaStorage)

	// Check Kafka connection
	err = kafkahandler.CheckAllKafkaConnections(config.FailedConsumeEnv.KAFKA_BROKERS)
	if err != nil {
		log.Fatal().Err(err).Msg("Error checking connection with Kafka")
	}

	// Initialize validator
	helper.InitializeValidator()

	// Create a WaitGroup to track worker goroutines
	var wg sync.WaitGroup
	// workDone channel waits for all workers to complete.
	workDone := make(chan int, 1)

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled on shutdown

	// Set up Kafka producers and consumers
	kafkahandler.InitializeKafkaProducerManager(config.FailedConsumeEnv.KAFKA_BROKERS)
	kafkahandler.InitializeKafkaConsumerManager(
		ctx,
		workDone,
		map[string]int{"failed-letter-queue": config.FailedConsumeEnv.KAFKA_FAILED_WORKERS},
		"consumer",
		&wg,
		config.FailedConsumeEnv.KAFKA_BROKERS,
		process.ProcessMessage)

	log.Info().Msg("media-docker-failed-consumer service started.")
	// Kafka consumers setup
	go kafkahandler.KafkaConsumer.KafkaConsumeSetup()

	// Shutdown handling using signal and worker tracking
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigChan:
			log.Info().Msgf("Received signal: %s. Shutting down...", sig)

			// Wait for all Kafka workers to finish before shutting down the service
			cancel() // Cancel context to signal Kafka workers to shut down
			log.Info().Msg("Waiting for Kafka workers to complete...")

			wg.Wait() // Wait for all worker goroutines to complete
			log.Info().Msg("All Kafka workers stopped")

			// ensure cleanUp happens
			cleanUpForConsumer()
			return

		case _, ok := <-workDone:
			if !ok {
				// If the channel is closed, all workers are done, so shut down
				log.Info().Msg("workDone channel closed, all Kafka workers finished. Initiating service shutdown...")

				// ensure cleanUp happens
				cleanUpForConsumer()
				return
			}
		}
	}
}

// cleanUpForConsumer performs final cleanup actions before shutdown
func cleanUpForConsumer() {
	if err := kafkahandler.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("Error while closing producer for media-docker-failed-consumer.")
	} else {
		log.Info().Msg("Producer closed for media-docker-failed-consumer.")
	}

	log.Info().Msg("media-docker-failed-consumer service shutdown complete.")
}
