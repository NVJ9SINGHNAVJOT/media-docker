package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducerManager defines a struct for managing Kafka operations.
// It holds a writer that is responsible for producing messages to Kafka topics.
type KafkaProducerManager struct {
	writer *kafka.Writer // The Kafka writer used for sending messages
}

// DLQMessage defines the structure of the message sent to the Dead-Letter Queue (DLQ).
type DLQMessage struct {
	OriginalTopic  string    `json:"originalTopic" validate:"required"`  // The original topic where the message came from
	Partition      int       `json:"partition" validate:"required"`      // Kafka partition of the original message
	Offset         int64     `json:"offset" validate:"required"`         // Offset of the original message
	HighWaterMark  int64     `json:"highWaterMark" validate:"required"`  // High watermark of the Kafka partition
	Value          string    `json:"value" validate:"required"`          // The original message value as a string
	ErrorDetails   string    `json:"errorDetails" validate:"required"`   // Details about the error encountered
	ProcessingTime time.Time `json:"processingTime" validate:"required"` // The timestamp when the message was processed
	ErrorTime      time.Time `json:"errorTime" validate:"required"`      // The timestamp when the error occurred
	Worker         string    `json:"worker" validate:"required"`         // The worker that processed the message
	CustomMessage  string    `json:"customMessage" validate:"required"`  // Any additional custom message
}

// NewKafkaProducerManager initializes and returns a new KafkaProducer instance.
// It accepts a slice of broker addresses and sets up the Kafka writer.
func NewKafkaProducerManager(brokers []string) *KafkaProducerManager {
	producer := &KafkaProducerManager{
		// Initialize the Kafka writer with the broker addresses and max attempts for message delivery
		writer: &kafka.Writer{
			Addr:        kafka.TCP(brokers...), // Set the address of Kafka brokers
			MaxAttempts: 10,                    // Number of delivery attempts in case of failure
		},
	}

	return producer // Return the initialized KafkaProducerManager
}

// Produce sends a message to the specified Kafka topic.
// It accepts the topic name and the message value to be sent.
func (kp *KafkaProducerManager) Produce(topic string, value interface{}) error {
	if kp.writer == nil {
		// Return an error if the Kafka producer has not been initialized
		return fmt.Errorf("kafka producer is not initialized")
	}

	// Marshal the value (interface) to JSON format for sending to Kafka
	jsonValue, err := json.Marshal(value)
	if err != nil {
		// Return an error if JSON marshaling fails
		return err
	}

	// Create the Kafka message with the specified topic and serialized JSON payload
	message := kafka.Message{
		Topic: topic,     // Specify the Kafka topic dynamically
		Value: jsonValue, // The serialized JSON payload
	}

	// Send the message to the Kafka topic
	err = kp.writer.WriteMessages(context.Background(), message)
	if err != nil {
		// Return an error if sending the message fails
		return err
	}

	return nil // Return nil if the message is successfully sent
}

// Close gracefully closes the Kafka producer.
// It ensures that resources are freed properly when the producer is no longer needed.
func (kp *KafkaProducerManager) Close() error {
	if kp.writer == nil {
		// Return an error if the Kafka producer has not been initialized
		return fmt.Errorf("kafka producer is not initialized")
	}

	// Close the Kafka writer to free resources and ensure all buffered messages are sent
	err := kp.writer.Close()
	if err != nil {
		// Return an error if closing the writer fails
		return err
	}

	kp.writer = nil // Set writer to nil to indicate it is closed
	return nil      // Return nil if the writer is closed successfully
}
