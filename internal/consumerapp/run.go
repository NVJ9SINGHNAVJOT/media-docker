// Package consumerapp is the shared runtime behind every media-docker consumer
// service.
//
// v4 splits the single combined consumer into one service per topic so that a
// backlog of video transcodes cannot starve image or audio work, and so each
// type scales on its own. Splitting the process, however, does not mean
// duplicating the process: everything the old entrypoints had in common --
// environment loading, logger setup, Kafka wiring, worker supervision, the
// shutdown handshake -- lives here, and each service supplies only the handful
// of values that make it different.
package consumerapp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Config describes one consumer service.
type Config struct {
	// Service is the service name, used in logs.
	Service string

	// EnvFile is the development environment file to load. In Docker the file is
	// absent and the container's environment is used instead.
	EnvFile string

	// Topic is the single Kafka topic this service consumes.
	Topic string

	// Handler processes one message payload and returns the asset id and a
	// description of the outcome. A non-nil error routes the message to the
	// dead-letter queue, where the failed consumer will retry it.
	//
	// Exactly one of Handler and RawHandler must be set.
	Handler func(payload []byte) (newId string, resMessage string, err error)

	// RawHandler processes a message without the dead-letter-queue wrapper, for
	// services whose failures must not be retried there: the delete consumer,
	// whose topic the failed consumer cannot replay, and the failed consumer
	// itself, which would otherwise re-queue into the topic it is draining.
	//
	// Exactly one of Handler and RawHandler must be set.
	RawHandler func(msg kafka.Message, workerName string)

	// RequiresFFmpeg marks a service that shells out to ffmpeg. Such a service
	// refuses to start without a working binary: otherwise it consumes happily
	// and dead-letters every single job, which hides the real cause.
	RequiresFFmpeg bool
}

// Run starts a consumer service and blocks until it has shut down.
func Run(cfg Config) {
	if (cfg.Handler == nil) == (cfg.RawHandler == nil) {
		panic("consumerapp: exactly one of Handler and RawHandler must be set")
	}

	// Load env file
	if err := pkg.LoadEnv(cfg.EnvFile); err != nil {
		fmt.Println("Error loading env file", err)
		panic(err)
	}

	// Validate environment variables
	if err := config.ValidateConsumerEnv(); err != nil {
		fmt.Println("Invalid environment variables", err)
		panic(err)
	}

	// Setup logger
	config.SetUpLogger(config.ConsumerEnv.ENVIRONMENT)

	// Verify ffmpeg before any worker starts, so a missing binary is one clear
	// startup failure rather than a dead-letter queue full of identical errors.
	if cfg.RequiresFFmpeg {
		version, err := pkg.CheckFFmpeg()
		if err != nil {
			log.Fatal().Err(err).Str("service", cfg.Service).Msg("ffmpeg is required but not usable")
		}
		log.Info().Str("ffmpeg", version).Msg("ffmpeg ready")
	}

	// Consumers write conversion output into asset directories that the server
	// has already created, so they only verify that the storage root is present.
	// They no longer touch the upload staging volume at all.
	pkg.DirExist(helper.Constants.MediaStorage)

	// Check Kafka connection. A service needs its own topic, and every service
	// using the standard Handler wrapper also produces failures to the DLQ.
	requiredTopics := []string{cfg.Topic}
	if cfg.Handler != nil && cfg.Topic != topics.FailedLetterQueue {
		requiredTopics = append(requiredTopics, topics.FailedLetterQueue)
	}
	if err := kafkahandler.CheckAllKafkaConnections(config.ConsumerEnv.KAFKA_BROKERS, requiredTopics...); err != nil {
		log.Fatal().Err(err).Str("service", cfg.Service).Msg("Error checking connection with Kafka")
	}

	// Initialize validator
	validator.InitializeValidator()

	// Create a WaitGroup to track worker goroutines
	var wg sync.WaitGroup
	// workDone channel waits for all workers to complete.
	workDone := make(chan int, 1)

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled on shutdown

	// Set up Kafka producers and consumers
	kafkahandler.InitializeKafkaProducerManager(config.ConsumerEnv.KAFKA_BROKERS)
	kafkahandler.InitializeKafkaConsumerManager(
		ctx,
		workDone,
		map[string]int{cfg.Topic: config.ConsumerEnv.KAFKA_WORKERS},
		&wg,
		config.ConsumerEnv.KAFKA_BROKERS,
		messageProcessor(cfg))

	log.Info().
		Str("topic", cfg.Topic).
		Int("workers", config.ConsumerEnv.KAFKA_WORKERS).
		Msgf("%s service started.", cfg.Service)

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
			cleanUp(cfg)
			return

		case _, ok := <-workDone:
			if !ok {
				// If the channel is closed, all workers are done, so shut down
				log.Info().Msg("workDone channel closed, all Kafka workers finished. Initiating service shutdown...")

				// ensure cleanUp happens
				cleanUp(cfg)
				return
			}
		}
	}
}

// cleanUp performs final cleanup actions before shutdown.
func cleanUp(cfg Config) {
	if err := kafkahandler.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msgf("Error while closing producer for %s.", cfg.Service)
	} else {
		log.Info().Msgf("Producer closed for %s.", cfg.Service)
	}

	// A consumer queues no background deletes: it only ever adds to an asset
	// directory, and it clears its own scratch directory synchronously. There is
	// nothing left to drain here.
	log.Info().Msgf("%s service shutdown complete.", cfg.Service)
}
