package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	consumerKafka "github.com/nvj9singhnavjot/media-docker/internal/media-docker-kafka-consumer/consumerKafka"
	"github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load env file
	err := pkg.LoadEnv(".env.consumer")
	if err != nil {
		fmt.Println("Error loading env file", err)
		panic(err)
	}

	// Validate environment variables
	err = config.ValidateKafkaConsumeEnv()
	if err != nil {
		fmt.Println("Invalid environment variables", err)
		panic(err)
	}

	// Setup logger
	config.SetUpLogger(config.KafkaConsumeEnv.ENVIRONMENT)

	config.CreateDirSetup()

	// Check Kafka connection
	err = kafka.CheckAllKafkaConnections(config.KafkaConsumeEnv.KAFKA_BROKERS)
	if err != nil {
		log.Fatal().Err(err).Msg("Error checking connection with Kafka")
	}

	// Initialize validator
	helper.InitializeValidator()

	// Create a WaitGroup to track worker goroutines
	var wg sync.WaitGroup

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled on shutdown

	// Error channel to listen to Kafka worker errors
	errChan := make(chan kafka.WorkerError)

	// Initialize WorkerTracker to track remaining workers per topic
	workerTracker := kafka.NewWorkerTracker(config.KafkaConsumeEnv.KAFKA_TOPIC_WORKERS)

	// Set up Kafka producers and consumers
	consumerKafka.KafkaProducer = kafka.NewKafkaProducerManager(config.KafkaConsumeEnv.KAFKA_BROKERS)
	consumerKafka.KafkaConsumer = kafka.NewKafkaConsumerManager(
		ctx, errChan, config.KafkaConsumeEnv.KAFKA_TOPIC_WORKERS,
		config.KafkaConsumeEnv.KAFKA_GROUP_PREFIX_ID, &wg,
		config.KafkaConsumeEnv.KAFKA_BROKERS, consumerKafka.ProcessMessage)

	// Start additional worker routines for deleting files and directories
	go pkg.DeleteFileWorker()
	go pkg.DeleteDirWorker()

	// Kafka consumers setup
	go consumerKafka.KafkaConsumer.KafkaConsumeSetup()

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
			pkg.CloseDeleteChannels()

			// Simulate a graceful shutdown delay, for deleting workers
			time.Sleep(10 * time.Second)

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

			log.Info().Msg("Kafka consumer service shutdown complete.")
			return

		case workerErr, ok := <-errChan:
			if !ok {
				// If the channel is closed, all workers are done, so shut down
				log.Info().Msg("Worker error channel closed, all Kafka workers finished. Initiating service shutdown...")
				pkg.CloseDeleteChannels()

				// Simulate a graceful shutdown delay, for deleting workers
				time.Sleep(10 * time.Second)
				log.Info().Msg("Kafka consumer service shutdown complete.")
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
}
