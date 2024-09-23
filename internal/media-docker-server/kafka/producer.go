package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
)

var RequestMap sync.Map
var writer *kafka.Writer

// InitializeKafkaProducer initializes the Kafka writer only once
func InitializeKafkaProducer(brokers []string) error {
	if writer != nil {
		// If the writer is already initialized, don't do it again
		return fmt.Errorf("kafka producer is already initialized")
	}

	// Initialize the Kafka writer with least-bytes balancer
	writer = &kafka.Writer{
		Addr:        kafka.TCP(brokers...),
		MaxAttempts: 5,
	}

	// Health check: validate the connection by attempting to retrieve metadata
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

// CloseKafkaProducer gracefully closes the Kafka producer
func CloseKafkaProducer() error {
	if writer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	err := writer.Close()
	if err != nil {
		return err
	}

	return nil
}

// ProduceKafkaMessage sends a message to the specified topic with 5 retry attempts
func ProduceKafkaMessage(topic string, value interface{}) error {
	// Marshal the value (interface) to JSON
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Create the Kafka message with key and value
	message := kafka.Message{
		Topic: topic,     // Specify the topic dynamically
		Value: jsonValue, // The serialized JSON message payload
	}

	// Write message with retry attempts handled by Kafka
	err = writer.WriteMessages(context.Background(), message)
	if err != nil {
		return err
	}

	return nil
}
