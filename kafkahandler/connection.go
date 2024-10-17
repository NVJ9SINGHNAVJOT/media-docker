package kafkahandler

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// CheckAllKafkaConnections checks if all Kafka brokers are reachable.
// It takes a slice of broker addresses as input and returns an error if any broker is not reachable.
func CheckAllKafkaConnections(brokers []string) error {
	// Create a new dialer for checking the connections
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second, // Timeout for dialing the broker
		KeepAlive: 20 * time.Second, // Keep connection alive duration
	}

	// Iterate over each broker and attempt to connect
	for _, broker := range brokers {
		// Attempt to dial the broker using TCP protocol
		conn, err := dialer.DialContext(context.Background(), "tcp", broker)
		if err != nil {
			// Return an error if the connection to the broker fails
			return fmt.Errorf("failed to connect to broker %s: %v", broker, err)
		}

		// Close the connection to free resources and check for any errors
		if closeErr := conn.Close(); closeErr != nil {
			// Return an error if closing the connection fails
			return fmt.Errorf("failed to close connection to broker %s: %v", broker, closeErr)
		}
	}

	// Return nil if all connections are successful, indicating all brokers are reachable
	return nil
}
