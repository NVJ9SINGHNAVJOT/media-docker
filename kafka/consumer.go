package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Retry settings
const retryAttempts = 5
const backoff = 2 * time.Second

// WorkerError to pass the topic and error
type WorkerError struct {
	Topic      string
	Err        error
	WorkerName string
}

// WorkerTracker keeps track of workers per topic and total workers
type workerTracker struct {
	workerCount map[string]int
	mu          sync.Mutex
}

// NewWorkerTracker initializes the worker tracker with the total worker count per topic
func NewWorkerTracker(workersPerGroup int, topics []string) *workerTracker {
	workerCount := make(map[string]int)

	// Initialize worker count per topic (workersPerGroup is workers per topic)
	for _, topic := range topics {
		workerCount[topic] = workersPerGroup
	}

	return &workerTracker{
		workerCount: workerCount,
	}
}

// DecrementWorker reduces the worker count for a topic and the total worker count,
// returning the remaining workers for that topic and a flag indicating if all workers are stopped.
func (w *workerTracker) DecrementWorker(topic string) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Decrease the worker count for the specific topic
	if count, exists := w.workerCount[topic]; exists && count > 0 {
		w.workerCount[topic]--
	}

	// Check if all workers are stopped by looking at the global count
	return w.workerCount[topic]
}

// KafkaConsumerManager manages the Kafka consumption setup
type KafkaConsumerManager struct {
	ctx             context.Context
	errChan         chan<- WorkerError
	workersPerTopic int
	groupID         string
	wg              *sync.WaitGroup
	topics          []string
	brokers         []string
	ProcessMessage  func(msg kafka.Message, workerName string) // Added field for the message processing function
}

// NewKafkaConsumerManager initializes the KafkaConsumerManager
func NewKafkaConsumerManager(ctx context.Context, errChan chan<- WorkerError, workersPerTopic int,
	groupID string, wg *sync.WaitGroup, topics, brokers []string,
	processMsg func(msg kafka.Message, workerName string)) *KafkaConsumerManager {
	return &KafkaConsumerManager{
		ctx:             ctx,
		errChan:         errChan,
		workersPerTopic: workersPerTopic,
		groupID:         groupID,
		wg:              wg,
		topics:          topics,
		brokers:         brokers,
		ProcessMessage:  processMsg, // Assign the function to the struct
	}
}

// KafkaConsumeSetup creates a single consumer group per topic and assigns workers within that group
func (k *KafkaConsumerManager) KafkaConsumeSetup() {
	// Iterate over each topic to create one group per topic and spawn workers
	for _, topic := range k.topics {
		// Create a single consumer group for each topic
		groupName := fmt.Sprintf(config.KafkaConsumeEnv.KAFKA_GROUP_PREFIX_ID+"-%s-group", topic)
		log.Info().Msgf("Starting consumer group: %s for topic: %s", groupName, topic)

		// Create workers for the current group and topic
		for workerID := 1; workerID <= k.workersPerTopic; workerID++ {
			k.wg.Add(1)
			go func(group string, topic string, workerID int) {
				defer k.wg.Done()
				workerName := fmt.Sprintf("%s-worker-%d", group, workerID)
				log.Info().Msgf("Starting worker: %s", workerName)
				if err := k.consumeWithRetry(group, topic, workerName); err != nil {
					k.errChan <- WorkerError{Topic: topic, Err: err, WorkerName: workerName}
				}
				log.Warn().Msgf("Shutting down worker: %s", workerName)
			}(groupName, topic, workerID)
		}
	}

	// Wait for all workers to finish
	k.wg.Wait()
	close(k.errChan)
}

// consumeWithRetry consumes messages and retries on failure
func (k *KafkaConsumerManager) consumeWithRetry(group, topic, workerName string) error {
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		err := k.consumeKafkaTopic(group, topic, workerName)

		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msgf("Error in %s, retrying (%d/%d)", workerName, attempt, retryAttempts)
			time.Sleep(backoff)
		} else {
			return nil // Successfully consumed
		}
	}
	return errors.New("retries exhausted for consuming")
}

// consumeKafkaTopic connects to Kafka and starts consuming messages for a specific topic with groupId.
func (k *KafkaConsumerManager) consumeKafkaTopic(group, topic, workerName string) error {
	// Create a new Kafka reader (consumer) with specified configuration, including brokers, group ID, and topic.
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.brokers, // Kafka brokers are retrieved from the environment config.
		GroupID: group,     // Consumer group ID for message sharing across workers.
		Topic:   topic,     // Topic from which messages will be consumed.
		Dialer: &kafka.Dialer{
			Timeout:   10 * time.Second, // Dial timeout for connecting to Kafka.
			KeepAlive: 5 * time.Minute,  // Keep connection alive for 5 minutes.
		},
		HeartbeatInterval: 3 * time.Second, // Interval for sending heartbeats to maintain the consumer group.
		MaxAttempts:       retryAttempts,   // Maximum number of retry attempts on failures.
	})

	// Ensure the Kafka reader is closed properly when the function exits.
	defer func() {
		if err := r.Close(); err != nil {
			log.Error().Err(err).Msgf("Failed to close Kafka reader for %s", workerName)
		}
	}()

	// Start an infinite loop to continuously fetch and process messages.
	for {
		select {
		case <-k.ctx.Done():
			// If the context is canceled (e.g., due to shutdown), log and exit the loop.
			log.Info().Msgf("Context cancelled, shutting down %s", workerName)
			return nil
		default:
			// Fetch a new message from the Kafka topic.
			msg, err := r.FetchMessage(k.ctx)
			if err != nil {
				return err // Return the error if fetching the message fails.
			}

			// Process the fetched message.
			k.ProcessMessage(msg, workerName)

			// After processing, commit the message offset to Kafka to mark it as consumed.
			if err := r.CommitMessages(k.ctx, msg); err != nil {
				log.Error().Err(err).Msgf("Failed to commit message in %s", workerName) // Log the error if commit fails.
			}
		}
	}
}
