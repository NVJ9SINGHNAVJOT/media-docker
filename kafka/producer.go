package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// KafkaProducerManager defines a struct for managing Kafka operations.
type KafkaProducerManager struct {
	writer *kafka.Writer
}

// NewKafkaProducer initializes and returns a new KafkaProducer instance.
func NewKafkaProducerManager(brokers []string) *KafkaProducerManager {
	producer := &KafkaProducerManager{
		// Initialize the Kafka writer
		writer: &kafka.Writer{
			Addr:        kafka.TCP(brokers...),
			MaxAttempts: 5,
		},
	}

	return producer
}

// Produce sends a message to the specified Kafka topic.
func (kp *KafkaProducerManager) Produce(topic string, value interface{}) error {
	if kp.writer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	// Marshal the value (interface) to JSON
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Create the Kafka message
	message := kafka.Message{
		Topic: topic,     // Specify the topic dynamically
		Value: jsonValue, // Serialized JSON payload
	}

	// Send the message to Kafka
	err = kp.writer.WriteMessages(context.Background(), message)
	if err != nil {
		return err
	}

	return nil
}

// Close gracefully closes the Kafka producer.
func (kp *KafkaProducerManager) Close() error {
	if kp.writer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	err := kp.writer.Close()
	if err != nil {
		return err
	}

	kp.writer = nil
	return nil
}
