package kafkahandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Global KafkaProducer variable represents the Kafka producer manager instance.
//
// Note: Before using KafkaProducer, the InitializeKafkaProducerManager function
// should be called to properly set up KafkaProducer with the kafkaProducerManager.
var KafkaProducer = kafkaProducerManager{}

// kafkaProducerManager handles Kafka operations, specifically producing messages to Kafka topics.
// It contains a Kafka writer that is responsible for writing messages to specified topics.
type kafkaProducerManager struct {
	writer *kafka.Writer // Kafka writer used for producing messages to Kafka topics
}

// InitializeKafkaProducerManager sets up the Kafka producer by creating a Kafka writer
// configured with the provided broker addresses. This initializes the kafkaProducerManager for message production.
func InitializeKafkaProducerManager(brokers []string) {
	// Create and configure the Kafka writer using the provided broker addresses
	KafkaProducer.writer = &kafka.Writer{
		Addr:        kafka.TCP(brokers...), // Address of Kafka brokers for message delivery
		MaxAttempts: 10,                    // Max retry attempts in case message delivery fails
	}
}

// Produce sends a message to the specified Kafka topic. The message value is marshaled to JSON format
// before being sent. It returns an error if the marshaling or writing process fails.
func (kp *kafkaProducerManager) Produce(topic string, value interface{}) error {
	// Convert the message value to JSON format for sending
	jsonValue, err := json.Marshal(value)
	if err != nil {
		// Return an error if the JSON marshaling fails
		return err
	}

	// Create a Kafka message with the topic and the marshaled JSON value
	message := kafka.Message{
		Topic: topic,     // The target Kafka topic to produce the message to
		Value: jsonValue, // Serialized message payload in JSON format
	}

	// Write the message to the Kafka topic
	return kp.writer.WriteMessages(context.Background(), message)
}

// Close gracefully shuts down the Kafka producer by closing the Kafka writer.
// It releases any resources associated with the producer, ensuring all buffered messages are sent.
func (kp *kafkaProducerManager) Close() error {
	// Check if the Kafka writer has been initialized before closing
	if kp.writer == nil {
		// Return an error if the producer is not initialized
		return fmt.Errorf("kafka producer is not initialized")
	}

	// Attempt to close the Kafka writer to free resources
	err := kp.writer.Close()
	if err != nil {
		// Return an error if the writer fails to close properly
		return err
	}

	// Mark the writer as closed by setting it to nil
	kp.writer = nil
	return nil // Return nil if the writer closed successfully
}
