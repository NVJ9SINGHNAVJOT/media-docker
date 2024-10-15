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
	// Configuration for the Kafka consumer
	KafkaConsumeEnv = kafkaConsumeConfig{}
	// Configuration for the failed consumer
	FailedConsumeEnv = failedConsumeConfig{}
)

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

// kafkaConsumeConfig holds the configuration settings for the Kafka consumer.
type kafkaConsumeConfig struct {
	ENVIRONMENT           string         // Current environment (e.g., development, production)
	KAFKA_GROUP_PREFIX_ID string         // Prefix for Kafka consumer group IDs
	KAFKA_BROKERS         []string       // List of Kafka broker addresses for message consumption
	KAFKA_TOPIC_WORKERS   map[string]int // Map of topics to the number of workers assigned for each topic
}

// failedConsumeConfig holds the configuration settings for the failed consumer.
type failedConsumeConfig struct {
	ENVIRONMENT          string   // Current environment (e.g., development, production)
	KAFKA_BROKERS        []string // List of Kafka broker addresses for handling failed messages
	KAFKA_FAILED_WORKERS int      // Number of workers assigned for processing failed messages
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

	clientPort, exists := os.LookupEnv("CLIENT_PORT")
	if !exists {
		return fmt.Errorf("client port is not provided")
	}

	ClientEnv.ENVIRONMENT = environment
	ClientEnv.ALLOWED_ORIGINS = strings.Split(allowedOrigins, ",")
	ClientEnv.CLIENT_PORT = clientPort

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

	// SERVER_PORT validation
	serverPort, exists := os.LookupEnv("SERVER_PORT")
	if !exists {
		return fmt.Errorf("server port is not provided")
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
	ServerEnv.SERVER_PORT = serverPort
	ServerEnv.BASE_URL = baseURL
	ServerEnv.KAFKA_BROKERS = strings.Split(brokers, ",")

	return nil
}

// ValidateKafkaConsumeEnv validates the environment variables for Kafka consume configuration.
func ValidateKafkaConsumeEnv() error {
	// Validate ENVIRONMENT
	environment, exists := os.LookupEnv("ENVIRONMENT")
	if !exists {
		return fmt.Errorf("environment is not provided")
	}

	// Validate KAFKA_GROUP_PREFIX_ID
	groupID, exists := os.LookupEnv("KAFKA_GROUP_PREFIX_ID")
	if !exists {
		return fmt.Errorf("kafka consume group ID is not provided")
	}

	// Validate KAFKA_BROKERS
	brokers, exists := os.LookupEnv("KAFKA_BROKERS")
	if !exists {
		return fmt.Errorf("kafka brokers are not provided")
	}

	// Validate worker counts for each Kafka topic
	topicWorkers := map[string]string{
		"video":             "KAFKA_VIDEO_WORKERS",
		"video-resolutions": "KAFKA_VIDEO_RESOLUTIONS_WORKERS",
		"image":             "KAFKA_IMAGE_WORKERS",
		"audio":             "KAFKA_AUDIO_WORKERS",
		"delete-file":       "KAFKA_DELETE_FILE_WORKERS",
	}

	workerCounts := make(map[string]int)

	for topic, envVar := range topicWorkers {
		workerCount, err := getAndValidateWorkerCount(envVar)
		if err != nil {
			return err
		}
		workerCounts[topic] = workerCount
	}

	// Set the validated environment variables in KafkaConsumeEnv
	KafkaConsumeEnv.ENVIRONMENT = environment
	KafkaConsumeEnv.KAFKA_GROUP_PREFIX_ID = groupID
	KafkaConsumeEnv.KAFKA_BROKERS = strings.Split(brokers, ",")
	KafkaConsumeEnv.KAFKA_TOPIC_WORKERS = workerCounts

	return nil
}

// ValidateFailedConsumeEnv validates the environment variables for Failed consume configuration.
func ValidateFailedConsumeEnv() error {
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

	// Validate KAFKA_FAILED_WORKERS
	workerCount, err := getAndValidateWorkerCount("KAFKA_FAILED_WORKERS")
	if err != nil {
		return err
	}

	// Set the validated environment variables in FailedConsumeEnv
	FailedConsumeEnv.ENVIRONMENT = environment
	FailedConsumeEnv.KAFKA_BROKERS = strings.Split(brokers, ",")
	FailedConsumeEnv.KAFKA_FAILED_WORKERS = workerCount

	return nil
}
