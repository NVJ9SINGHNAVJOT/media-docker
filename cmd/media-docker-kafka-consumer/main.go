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
	// workDone channel waits for all workers to complete.
	workDone := make(chan int, 1)

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled on shutdown

	// Set up Kafka producers and consumers
	consumerKafka.KafkaProducer = kafka.NewKafkaProducerManager(config.KafkaConsumeEnv.KAFKA_BROKERS)
	consumerKafka.KafkaConsumer = kafka.NewKafkaConsumerManager(
		ctx, workDone, config.KafkaConsumeEnv.KAFKA_TOPIC_WORKERS,
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
	if err := consumerKafka.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("Error while closing producer for media-docker-consumer")
	} else {
		log.Info().Msg("Producer closed for media-docker-consumer.")
	}

	pkg.CloseDeleteChannels()
	log.Info().Msg("Delete channels closed.")

	// Simulate a graceful shutdown delay, for deleting workers
	time.Sleep(5 * time.Second)
	log.Info().Msg("Kafka consumer service shutdown complete.")
}
