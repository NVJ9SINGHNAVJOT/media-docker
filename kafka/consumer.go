package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Retry settings for Kafka message consumption.
const retryAttempts = 5         // Number of retry attempts for consuming messages
const backoff = 2 * time.Second // Duration to wait before retrying after a failure

// WorkerError is a struct used to pass error information, including the topic and worker details.
type WorkerError struct {
	Topic      string // The Kafka topic related to the error
	Err        error  // The error encountered
	WorkerName string // The name of the worker that encountered the error
}

// workerTracker keeps track of the number of workers assigned to each Kafka topic.
type workerTracker struct {
	workerCount map[string]int // A map that holds the count of workers per topic
	mu          sync.Mutex     // Mutex for thread-safe access to worker counts
}

// NewWorkerTracker initializes a worker tracker for managing the count of workers per topic.
func NewWorkerTracker(workersPerTopic map[string]int) *workerTracker {
	workerCount := make(map[string]int)

	// Initialize worker count for each topic based on the number of workers specified per topic.
	for topic, count := range workersPerTopic {
		workerCount[topic] = count
	}

	return &workerTracker{
		workerCount: workerCount, // Return a tracker with initialized worker counts
	}
}

// DecrementWorker reduces the worker count for a specified topic and checks if all workers are stopped.
// It returns the remaining workers for that topic.
func (w *workerTracker) DecrementWorker(topic string) int {
	w.mu.Lock()         // Lock to ensure thread-safe access
	defer w.mu.Unlock() // Unlock when function exits

	// Reduce the count of workers for the specified topic if it exists and is greater than zero.
	if count, exists := w.workerCount[topic]; exists && count > 0 {
		w.workerCount[topic]--
	}

	// Return the updated worker count for the topic.
	return w.workerCount[topic]
}

// KafkaConsumerManager manages the Kafka consumption setup, including context, error channel, and worker configuration.
type KafkaConsumerManager struct {
	ctx             context.Context                            // Context for cancellation and timeouts
	errChan         chan<- WorkerError                         // Channel for passing error information
	workersPerTopic map[string]int                             // Map of topic to number of workers
	groupID         string                                     // Consumer group ID for coordinating workers
	wg              *sync.WaitGroup                            // WaitGroup for synchronizing goroutines
	brokers         []string                                   // List of Kafka broker addresses
	ProcessMessage  func(msg kafka.Message, workerName string) // Function for processing consumed messages
}

// NewKafkaConsumerManager initializes a new KafkaConsumerManager with the necessary parameters.
func NewKafkaConsumerManager(ctx context.Context, errChan chan<- WorkerError, workersPerTopic map[string]int,
	groupID string, wg *sync.WaitGroup, brokers []string,
	processMsg func(msg kafka.Message, workerName string)) *KafkaConsumerManager {
	return &KafkaConsumerManager{
		ctx:             ctx,
		errChan:         errChan,
		workersPerTopic: workersPerTopic,
		groupID:         groupID,
		wg:              wg,
		brokers:         brokers,
		ProcessMessage:  processMsg, // Assign the message processing function to the struct
	}
}

// KafkaConsumeSetup creates a consumer group for each topic and spawns workers within that group.
func (k *KafkaConsumerManager) KafkaConsumeSetup() {
	// Iterate over each topic to create a consumer group and spawn workers.
	for topic, workersCount := range k.workersPerTopic {
		// Create a consumer group name for each topic.
		groupName := fmt.Sprintf(k.groupID+"-%s-group", topic)
		// Log the start of a new consumer group
		log.Info().
			Str("group", groupName).
			Str("topic", topic).
			Int("workersCount", workersCount).
			Msg("Starting consumer group")

		// Create workers for the current consumer group and topic.
		for workerID := 1; workerID <= workersCount; workerID++ {
			k.wg.Add(1) // Increment the WaitGroup counter
			go func(group string, topic string, workerID int) {
				defer k.wg.Done()                                          // Decrement the counter when the goroutine completes
				workerName := fmt.Sprintf("%s-worker-%d", group, workerID) // Name for the worker
				// Log worker start
				log.Info().
					Str("worker", workerName).
					Str("topic", topic).
					Msg("Starting worker")

				// Start consuming messages with retry logic
				if err := k.consumeWithRetry(group, topic, workerName); err != nil {
					// Send error details to the error channel if consumption fails
					k.errChan <- WorkerError{Topic: topic, Err: err, WorkerName: workerName}
				}
				// Log worker shutdown
				log.Warn().
					Str("worker", workerName).
					Str("topic", topic).
					Msg("Shutting down worker")
			}(groupName, topic, workerID) // Pass arguments to the goroutine
		}
	}

	// Wait for all workers to finish processing
	k.wg.Wait()
	// Log completion of all workers
	log.Info().Msg("All workers have completed processing")

	log.Info().Msg("Closing Worker error channel")
	// Close the error channel after all workers are done
	close(k.errChan)
}

// consumeWithRetry attempts to consume messages and retries on failure.
func (k *KafkaConsumerManager) consumeWithRetry(group, topic, workerName string) error {
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		err := k.consumeKafkaTopic(group, topic, workerName) // Attempt to consume messages

		if err != nil && !errors.Is(err, context.Canceled) {
			// Log the error and retry if not cancelled
			log.Error().
				Err(err).
				Str("worker", workerName).
				Str("topic", topic).
				Str("attempt", fmt.Sprintf("%d/%d", attempt, retryAttempts)).
				Msg("Error during message consumption, retrying")
			time.Sleep(backoff) // Wait before retrying
		} else {
			// Log successful consumption
			log.Info().
				Str("worker", workerName).
				Str("topic", topic).
				Msg("Successfully consumed messages")
			return nil // Successfully consumed messages
		}
	}
	return errors.New("retries exhausted for consuming") // Return error if retries are exhausted
}

// consumeKafkaTopic connects to Kafka and starts consuming messages for a specific topic with the given group ID.
func (k *KafkaConsumerManager) consumeKafkaTopic(group, topic, workerName string) error {
	// Create a new Kafka reader (consumer) with specified configuration.
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.brokers, // List of Kafka brokers
		GroupID: group,     // Consumer group ID
		Topic:   topic,     // Topic to consume messages from
		Dialer: &kafka.Dialer{
			Timeout:   10 * time.Second, // Dial timeout for Kafka connection
			KeepAlive: 5 * time.Minute,  // Keep connection alive for 5 minutes
		},
		HeartbeatInterval: 3 * time.Second, // Heartbeat interval to maintain the consumer group
		MaxAttempts:       retryAttempts,   // Max number of retries for message consumption
	})

	// Ensure the Kafka reader is closed properly when the function exits.
	defer func() {
		if err := r.Close(); err != nil {
			log.Error().Err(err).Msgf("Failed to close Kafka reader for %s", workerName) // Log error if closing fails
		}
	}()

	// Start an infinite loop to continuously fetch and process messages.
	for {
		select {
		case <-k.ctx.Done():
			// If the context is cancelled, log and exit the loop.
			log.Info().Msgf("Context cancelled, shutting down %s", workerName)
			return nil
		default:
			// Fetch a new message from the Kafka topic.
			msg, err := r.FetchMessage(k.ctx)
			if err != nil {
				return err // Return error if fetching the message fails.
			}

			// Process the fetched message using the provided processing function.
			k.ProcessMessage(msg, workerName)

			// Commit the message offset to Kafka to mark it as consumed.
			if err := r.CommitMessages(k.ctx, msg); err != nil {
				// Log error if commit fails.
				log.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Str("worker", workerName).
					Str("kafka_message", string(msg.Value))
			}
		}
	}
}
