package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type clientConfig struct {
	ENVIRONMENT     string
	ALLOWED_ORIGINS []string
	CLIENT_PORT     string
}

type serverConfig struct {
	ENVIRONMENT           string
	ALLOWED_ORIGINS       []string
	SERVER_KEY            string
	KAFKA_GROUP_WORKERS   int
	KAFKA_GROUP_PREFIX_ID string
	KAFKA_BROKERS         []string
	BASE_URL              string
	SERVER_PORT           string
}

type kafkaConsumeConfig struct {
	ENVIRONMENT           string
	KAFKA_GROUP_WORKERS   int
	KAFKA_GROUP_PREFIX_ID string
	KAFKA_BROKERS         []string
}

var ClientEnv = clientConfig{}
var ServerEnv = serverConfig{}
var KafkaConsumeEnv = kafkaConsumeConfig{}

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
	environment, exists := os.LookupEnv("ENVIRONMENT")
	if !exists {
		return fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exists := os.LookupEnv("ALLOWED_ORIGINS_SERVER")
	if !exists {
		return fmt.Errorf("allowed origins are not provided")
	}

	serverKey, exists := os.LookupEnv("SERVER_KEY")
	if !exists {
		return fmt.Errorf("server key is not provided")
	}

	groupWorkersStr, exists := os.LookupEnv("KAFKA_GROUP_WORKERS")
	if !exists {
		return fmt.Errorf("kafka consume group workers are not provided")
	}

	groupWorkers, err := strconv.Atoi(groupWorkersStr)
	if err != nil {
		return fmt.Errorf("invalid Kafka consume group workers size: %v", err)
	}

	if groupWorkers < 1 {
		return fmt.Errorf("minimum size required for Kafka consume group workers is 1")
	}

	groupID, exists := os.LookupEnv("KAFKA_GROUP_PREFIX_ID")
	if !exists {
		return fmt.Errorf("kafka consume group ID is not provided")
	}

	brokers, exists := os.LookupEnv("KAFKA_BROKERS")
	if !exists {
		return fmt.Errorf("kafka brokers are not provided")
	}

	serverPort, exists := os.LookupEnv("SERVER_PORT")
	if !exists {
		return fmt.Errorf("server port is not provided")
	}

	baseURL, exists := os.LookupEnv("BASE_URL")
	if !exists {
		return fmt.Errorf("base URL is not provided")
	}

	ServerEnv.ENVIRONMENT = environment
	ServerEnv.ALLOWED_ORIGINS = strings.Split(allowedOrigins, ",")
	ServerEnv.SERVER_KEY = serverKey
	ServerEnv.SERVER_PORT = serverPort
	ServerEnv.BASE_URL = baseURL
	ServerEnv.KAFKA_GROUP_WORKERS = groupWorkers
	ServerEnv.KAFKA_GROUP_PREFIX_ID = groupID
	ServerEnv.KAFKA_BROKERS = strings.Split(brokers, ",")

	return nil
}

// ValidateKafkaConsumeEnv validates the environment variables for Kafka consume configuration.
func ValidateKafkaConsumeEnv() error {
	environment, exists := os.LookupEnv("ENVIRONMENT")
	if !exists {
		return fmt.Errorf("environment is not provided")
	}

	groupWorkersStr, exists := os.LookupEnv("KAFKA_GROUP_WORKERS")
	if !exists {
		return fmt.Errorf("kafka consume group workers are not provided")
	}

	groupWorkers, err := strconv.Atoi(groupWorkersStr)
	if err != nil {
		return fmt.Errorf("invalid Kafka consume group workers size: %v", err)
	}

	if groupWorkers < 1 {
		return fmt.Errorf("minimum size required for Kafka consume group workers is 1")
	}

	groupID, exists := os.LookupEnv("KAFKA_GROUP_PREFIX_ID")
	if !exists {
		return fmt.Errorf("kafka consume group ID is not provided")
	}

	brokers, exists := os.LookupEnv("KAFKA_BROKERS")
	if !exists {
		return fmt.Errorf("kafka brokers are not provided")
	}

	KafkaConsumeEnv.ENVIRONMENT = environment
	KafkaConsumeEnv.KAFKA_GROUP_WORKERS = groupWorkers
	KafkaConsumeEnv.KAFKA_GROUP_PREFIX_ID = groupID
	KafkaConsumeEnv.KAFKA_BROKERS = strings.Split(brokers, ",")

	return nil
}
