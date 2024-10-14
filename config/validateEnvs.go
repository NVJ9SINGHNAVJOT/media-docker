package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

type clientConfig struct {
	ENVIRONMENT     string
	ALLOWED_ORIGINS []string
	CLIENT_PORT     string
}

type serverConfig struct {
	ENVIRONMENT     string
	ALLOWED_ORIGINS []string
	SERVER_KEY      string
	KAFKA_BROKERS   []string
	BASE_URL        string
	SERVER_PORT     string
}

type kafkaConsumeConfig struct {
	ENVIRONMENT           string
	KAFKA_GROUP_PREFIX_ID string
	KAFKA_BROKERS         []string
	KAFKA_TOPIC_WORKERS   map[string]int
}

type failedConsumeConfig struct {
	ENVIRONMENT   string
	KAFKA_BROKERS []string
	WORKERS       int
}

var ClientEnv = clientConfig{}
var ServerEnv = serverConfig{}
var KafkaConsumeEnv = kafkaConsumeConfig{}
var FailedConsumeEnv = failedConsumeConfig{}

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
//
// NOTE: By default, 2 workers are set. This can be increased by providing a workers count as a parameter.
func ValidateFailedConsumeEnv(workers ...int) error {
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

	// Set the workers count, use default of 2 if not provided
	workerCount := 2
	if len(workers) > 0 {
		if workers[0] < 1 {
			return fmt.Errorf("workers count must be at least 1")
		}
		workerCount = workers[0]
	}

	// Set the validated environment variables in FailedConsumeEnv
	FailedConsumeEnv.ENVIRONMENT = environment
	FailedConsumeEnv.KAFKA_BROKERS = strings.Split(brokers, ",")
	FailedConsumeEnv.WORKERS = workerCount

	return nil
}
