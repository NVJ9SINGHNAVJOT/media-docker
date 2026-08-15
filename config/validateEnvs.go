package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Declare instances of the configuration structs for different environments
var (
	// Configuration for the media-docker-client
	ClientEnv = clientConfig{}
	// Configuration for the media-docker-server
	ServerEnv = serverConfig{}
	// Configuration shared by every consumer service
	ConsumerEnv = consumerConfig{}
)

// clientConfig holds the configuration settings for the media-docker-client.
type clientConfig struct {
	ENVIRONMENT     string   // Current environment (e.g., development, production)
	ALLOWED_ORIGINS []string // List of allowed origins for CORS to restrict access
	CLIENT_PORT     string   // Port on which the client service will run
}

// serverConfig holds the configuration settings for the media-docker-server.
type serverConfig struct {
	ENVIRONMENT     string   // Current environment (e.g., development, production)
	ALLOWED_ORIGINS []string // List of allowed origins for CORS to restrict access
	SERVER_KEY      string   // Authentication key for server communication
	KAFKA_BROKERS   []string // List of Kafka broker addresses for message processing
	BASE_URL        string   // Base URL for client access to media files
	SERVER_PORT     string   // Port on which the server will run
}

// consumerConfig holds the configuration settings shared by every consumer service.
//
// Since v4 each consumer owns exactly one topic, so a single worker count
// replaces the per-topic map the combined consumer needed.
type consumerConfig struct {
	ENVIRONMENT   string   // Current environment (e.g., development, production)
	KAFKA_BROKERS []string // List of Kafka broker addresses for message consumption
	KAFKA_WORKERS int      // Number of workers for this service's topic
}

// getAndValidateWorkerCount retrieves and validates worker count from environment variables.
// It checks that the worker count is not below 1, otherwise returns an error.
func getAndValidateWorkerCount(envVar string) (int, error) {
	workerCountStr, exists := os.LookupEnv(envVar)
	if !exists {
		return 0, fmt.Errorf("%s is not provided", envVar)
	}

	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return 0, fmt.Errorf("invalid worker count for %s: %v", envVar, err)
	}

	if workerCount < 1 {
		return 0, fmt.Errorf("minimum size required for %s is 1", envVar)
	}

	return workerCount, nil
}

// ValidateClientEnv validates the environment variables for the client configuration.
func ValidateClientEnv() error {
	environment, exists := os.LookupEnv("ENVIRONMENT")
	if !exists {
		return fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exists := os.LookupEnv("ALLOWED_ORIGINS_CLIENT")
	if !exists {
		return fmt.Errorf("allowed origins are not provided")
	}

	ClientEnv.ENVIRONMENT = environment
	ClientEnv.ALLOWED_ORIGINS = strings.Split(allowedOrigins, ",")
	ClientEnv.CLIENT_PORT = "7000"

	return nil
}

// ValidateServerEnv validates the environment variables for the server configuration.
func ValidateServerEnv() error {
	// ENVIRONMENT validation
	environment, exists := os.LookupEnv("ENVIRONMENT")
	if !exists {
		return fmt.Errorf("environment is not provided")
	}

	// ALLOWED_ORIGINS_SERVER validation
	allowedOrigins, exists := os.LookupEnv("ALLOWED_ORIGINS_SERVER")
	if !exists {
		return fmt.Errorf("allowed origins are not provided")
	}

	// SERVER_KEY validation
	serverKey, exists := os.LookupEnv("SERVER_KEY")
	if !exists {
		return fmt.Errorf("server key is not provided")
	}

	// KAFKA_BROKERS validation
	brokers, exists := os.LookupEnv("KAFKA_BROKERS")
	if !exists {
		return fmt.Errorf("kafka brokers are not provided")
	}

	// BASE_URL validation
	baseURL, exists := os.LookupEnv("BASE_URL")
	if !exists {
		return fmt.Errorf("base URL is not provided")
	}

	// Populate the ServerEnv struct
	ServerEnv.ENVIRONMENT = environment
	ServerEnv.ALLOWED_ORIGINS = strings.Split(allowedOrigins, ",")
	ServerEnv.SERVER_KEY = serverKey
	ServerEnv.SERVER_PORT = "7007"
	ServerEnv.BASE_URL = baseURL
	ServerEnv.KAFKA_BROKERS = strings.Split(brokers, ",")

	return nil
}

// ValidateConsumerEnv validates the environment variables every consumer service needs.
//
// CAUTION: KAFKA_WORKERS is bounded by the partition count of the service's
// topic, summed across every running instance of that service. Workers beyond
// that count are never assigned a partition and sit idle. Partition counts are
// declared in ./kafka_config.sh.
func ValidateConsumerEnv() error {
	// Validate ENVIRONMENT
	environment, exists := os.LookupEnv("ENVIRONMENT")
	if !exists {
		return fmt.Errorf("environment is not provided")
	}

	// Validate KAFKA_BROKERS
	brokers, exists := os.LookupEnv("KAFKA_BROKERS")
	if !exists {
		return fmt.Errorf("kafka brokers are not provided")
	}

	// Validate KAFKA_WORKERS
	workerCount, err := getAndValidateWorkerCount("KAFKA_WORKERS")
	if err != nil {
		return err
	}

	// Set the validated environment variables in ConsumerEnv
	ConsumerEnv.ENVIRONMENT = environment
	ConsumerEnv.KAFKA_BROKERS = strings.Split(brokers, ",")
	ConsumerEnv.KAFKA_WORKERS = workerCount

	return nil
}
