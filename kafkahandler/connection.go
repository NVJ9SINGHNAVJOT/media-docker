package kafkahandler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

const (
	// connectRetryAttempts is how many times startup waits for Kafka before giving up.
	connectRetryAttempts = 5

	// connectRetryBackoff is the pause between connection attempts.
	connectRetryBackoff = 4 * time.Second
)

// CheckAllKafkaConnections verifies that every broker is reachable and that each
// required topic exists, retrying a few times so a broker that is merely slow to
// come up does not kill the service.
//
// A service calls this before it starts any worker, so a misconfigured cluster
// fails loudly at boot instead of silently failing every message afterwards.
func CheckAllKafkaConnections(brokers []string, requiredTopics ...string) error {
	var lastErr error

	for attempt := 1; attempt <= connectRetryAttempts; attempt++ {
		lastErr = checkKafkaOnce(brokers, requiredTopics)
		if lastErr == nil {
			log.Info().
				Strs("brokers", brokers).
				Strs("topics", requiredTopics).
				Int("attempts", attempt).
				Msg("Kafka connection established")
			return nil
		}

		if attempt < connectRetryAttempts {
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("of", connectRetryAttempts).
				Str("backoff", connectRetryBackoff.String()).
				Msg("Kafka not ready, retrying")
			time.Sleep(connectRetryBackoff)
		}
	}

	return fmt.Errorf("kafka not ready after %d attempts: %w", connectRetryAttempts, lastErr)
}

// checkKafkaOnce performs a single connection and topic-existence check.
func checkKafkaOnce(brokers []string, requiredTopics []string) error {
	// Create a new dialer for checking the connections
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second, // Timeout for dialing the broker
		KeepAlive: 20 * time.Second, // Keep connection alive duration
	}

	var metadataChecked bool

	// Iterate over each broker and attempt to connect
	for _, broker := range brokers {
		// Attempt to dial the broker using TCP protocol
		conn, err := dialer.DialContext(context.Background(), "tcp", broker)
		if err != nil {
			// Return an error if the connection to the broker fails
			return fmt.Errorf("failed to connect to broker %s: %v", broker, err)
		}

		// Cluster metadata is identical on every broker, so the topics are
		// verified once, against the first broker that answers.
		if !metadataChecked && len(requiredTopics) > 0 {
			topicErr := checkTopicsExist(conn, requiredTopics)
			if topicErr != nil {
				conn.Close()
				return topicErr
			}
			metadataChecked = true
		}

		// Close the connection to free resources and check for any errors
		if closeErr := conn.Close(); closeErr != nil {
			// Return an error if closing the connection fails
			return fmt.Errorf("failed to close connection to broker %s: %v", broker, closeErr)
		}
	}

	return nil
}

// checkTopicsExist reads cluster metadata and reports every required topic that
// is missing.
func checkTopicsExist(conn *kafka.Conn, requiredTopics []string) error {
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("failed to read kafka cluster metadata: %v", err)
	}

	existing := make(map[string]bool, len(partitions))
	for _, p := range partitions {
		existing[p.Topic] = true
	}

	var missing []string
	for _, topic := range requiredTopics {
		if !existing[topic] {
			missing = append(missing, topic)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf(
			"kafka topics missing: %s (topics are defined in ./kafka_config.sh, create them with `task kafka-topics`)",
			strings.Join(missing, ", "),
		)
	}

	return nil
}
