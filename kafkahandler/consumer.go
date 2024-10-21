package kafkahandler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Global KafkaConsumer variable represents the Kafka consumer manager instance.
//
// Note: Before using KafkaConsumer, the InitializeKafkaConsumerManager function
// should be called to properly set up KafkaConsumer with the kafkaConsumerManager.
var KafkaConsumer = kafkaConsumerManager{}

// Retry settings for Kafka message consumption.
const retryAttempts = 5         // Number of retry attempts for consuming messages
const backoff = 4 * time.Second // Duration to wait before retrying after a failure

// topicTracker tracks the number of active workers for a specific topic.
// It ensures thread-safe access to the worker count using a mutex.
type topicTracker struct {
	count int        // Number of active workers for the topic
	lock  sync.Mutex // Mutex to synchronize access to the worker count
}

// workerTracker manages the worker count for multiple topics using a map.
// Each topic has its own tracker to maintain and control worker concurrency.
type workerTracker struct {
	topics map[string]*topicTracker // A map of topics with their respective worker trackers
}

// kafkaConsumerManager oversees the configuration and management of Kafka consumers,
// including context handling, signaling channels, and worker settings.
type kafkaConsumerManager struct {
	ctx                context.Context                            // Context for managing cancellation and timeouts.
	workDone           chan int                                   // Channel for signaling when all workers have finished processing messages.
	workersPerTopic    map[string]int                             // Mapping of topics to their corresponding worker counts.
	wg                 *sync.WaitGroup                            // WaitGroup to synchronize the completion of goroutines.
	brokers            []string                                   // Slice of Kafka broker addresses to connect to.
	ProcessMessage     func(msg kafka.Message, workerName string) // Function to handle incoming Kafka messages.
	workerErrorManager workerTracker                              // Instance of workerTracker for monitoring and managing worker statuses.
}

// InitializeKafkaConsumerManager sets up the kafkaConsumerManager instance.
// It configures the necessary parameters for managing Kafka consumers based on the provided input.
//
// Parameters:
//   - ctx: The context.Context used for managing cancellation and timeouts for the consumer manager.
//   - workDone: A channel of type int that signals when all worker goroutines have completed their tasks.
//   - workersPerTopic: A map where keys are topic names (strings) and values are the number of workers (ints)
//     assigned to each topic, dictating how many consumer instances will process messages from each topic.
//   - wg: A pointer to a sync.WaitGroup that helps synchronize the completion of goroutines, allowing
//     the main program to wait for all worker goroutines to finish before proceeding or exiting.
//   - brokers: A slice of strings representing the addresses of the Kafka brokers to which the consumer manager will connect.
//   - processMsg: A function that takes a kafka.Message and a workerName (string) as parameters. This function is
//     called to process each message that the consumer receives.
func InitializeKafkaConsumerManager(
	ctx context.Context,
	workDone chan int,
	workersPerTopic map[string]int,
	wg *sync.WaitGroup, brokers []string,
	processMsg func(msg kafka.Message, workerName string)) {

	// Initialize the workerErrorManager with a map to track the state of each topic.
	workerErrorManager := workerTracker{
		topics: make(map[string]*topicTracker), // Create a map for tracking workers per topic.
	}

	// Set up a topic tracker for each topic and initialize its worker count.
	for topic, workersCount := range workersPerTopic {
		workerErrorManager.topics[topic] = &topicTracker{count: workersCount} // Assign the worker count to the topic tracker.
	}

	// Configure the kafkaConsumerManager instance with the provided parameters.
	KafkaConsumer = kafkaConsumerManager{
		ctx:                ctx,                // Context for managing cancellation and timeouts.
		workDone:           workDone,           // Channel to signal when all workers have completed their tasks.
		workersPerTopic:    workersPerTopic,    // Mapping of topics to the number of workers assigned to each.
		wg:                 wg,                 // WaitGroup for synchronizing goroutines.
		brokers:            brokers,            // List of Kafka broker addresses to connect to.
		ProcessMessage:     processMsg,         // Function to process consumed Kafka messages.
		workerErrorManager: workerErrorManager, // Instance to track worker statuses for error management.
	}
}

// decrementWorker safely decreases the worker count for a given topic in the workerTracker.
// It locks the topic-specific mutex to ensure that no other goroutines modify the worker count simultaneously.
//
// If there are remaining workers for the topic, it logs a warning with the updated worker count.
// If all workers for the topic are exhausted, it logs an error indicating that no workers remain.
func (k *kafkaConsumerManager) decrementWorker(group, topic, workerName string) {
	// Acquire the lock for the given topic to ensure thread-safe access to the worker count
	t := k.workerErrorManager.topics[topic]
	t.lock.Lock()
	defer t.lock.Unlock() // Ensure the lock is released after the count is updated

	// Decrement the worker count for the specified topic
	t.count--
	log.Warn().
		Str("topic", topic).
		Str("group", group).
		Str("workerName", workerName).Msg("worker stopped")

	// HACK: New worker can be added if needed.
	// Currently, no new worker is started, but all necessary parameters (group name, topic name, worker name)
	// are available to start a replacement worker if required after this decrement operation.
	if t.count > 0 {
		// Log a warning if only one or a few workers remain for the topic
		log.Warn().
			Str("topic", topic).
			Str("group", group).
			Msgf("Only %d worker(s) remaining", t.count)
	} else {
		// Log an error if no workers are left to process the topic
		log.Error().
			Str("topic", topic).
			Str("group", group).
			Msg("No workers remaining")
	}
}

// KafkaConsumeSetup creates a consumer group for each topic and spawns workers within that group.
//
// NOTE: It is important to call this function within a goroutine, as it waits for all workers to complete.
func (k *kafkaConsumerManager) KafkaConsumeSetup() {

	// Iterate over each topic to create a consumer group and spawn the corresponding workers.
	for topic, workersCount := range k.workersPerTopic {
		// Create a Kafka consumer group name by combining "consumer" as a prefix,
		// the topic name, and "group" as a suffix to indicate it is a consumer group configuration for the topic.
		// For example, if the topic is "video",
		// the resulting groupName will be: "consumer-video-group".
		// This "consumer-video-group" is the group name used for all workers under this topic.
		groupName := fmt.Sprintf("consumer-%s-group", topic)

		// Log the creation of the consumer group, showing details such as the topic name,
		// the generated group name, and the number of workers assigned to handle this topic's messages.
		log.Info().
			Str("topic", topic).
			Str("group", groupName).
			Int("workersCount", workersCount).
			Msg("Starting consumer group")

		// Launch the specified number of workers to handle message consumption for the current topic.
		// Each worker will operate concurrently, allowing for parallel processing of messages.
		for workerID := 1; workerID <= workersCount; workerID++ {
			k.wg.Add(1) // Increment the WaitGroup counter to track the completion of each worker.

			// Create a unique worker name by appending a worker-specific ID to the group name.
			// This ensures that each worker within the group has a unique identity, which includes the topic name,
			// the group name, and its own ID. For example:
			// If the groupName is "consumer-video-group" and the workerID is 1,
			// the resulting workerName will be: "consumer-video-group-worker-1".
			// Here, "consumer-video-group" is the group name for this worker, and "worker-1" is the worker's unique ID within this group.
			workerName := fmt.Sprintf("%s-worker-%d", groupName, workerID)

			// Start a goroutine for each worker, which will handle message consumption with retry logic.
			// This allows each worker to process messages independently and concurrently.
			go k.consumeWithRetry(groupName, topic, workerName)
		}
	}

	log.Info().
		Interface("workers", k.workersPerTopic).
		Msg("All workers have started sucessfully")

	// Wait for all workers to finish processing before proceeding.
	k.wg.Wait()
	log.Info().Msg("All workers have completed processing") // Log the completion of all workers.

	// Close the workDone channel to signal that worker processing is complete.
	// This informs other components of the system that all tasks related to consumption have concluded.
	log.Info().Msg("Closing workDone channel")
	close(k.workDone)
}

// consumeWithRetry attempts to consume messages from a Kafka topic and retries upon failure.
// This function will make up to 5 retry attempts before giving up.
func (k *kafkaConsumerManager) consumeWithRetry(group, topic, workerName string) {
	defer k.wg.Done() // Decrement the WaitGroup counter when the worker finishes its task.

	// Log the start of the worker, providing context on the topic and group it is processing.
	log.Info().
		Str("topic", topic).
		Str("group", group).
		Str("worker", workerName).
		Msg("Starting worker")

	// Loop to attempt message consumption with retries on failure.
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		// Try to consume a message from the specified Kafka topic.
		err := k.consumeKafkaTopic(group, topic, workerName)

		// Check if an error occurred during consumption.
		// Retry if the error is not due to context cancellation.
		if err != nil && !errors.Is(err, context.Canceled) {
			// Log the encountered error and the current retry attempt number.
			log.Error().
				Err(err).
				Str("worker", workerName).
				Str("attempt", fmt.Sprintf("%d/%d", attempt, retryAttempts)).
				Msg("Error during message consumption, retrying")

			// If this is not the last retry attempt, apply a backoff duration before the next attempt.
			if attempt < retryAttempts {
				time.Sleep(backoff)
			}
		} else {
			// If message consumption is successful, log the success and exit the retry loop.
			log.Info().
				Str("worker", workerName).
				Msg("Successfully consumed messages")
			return
		}
	}

	// If all retry attempts are exhausted, log a warning and decrement the worker count.
	log.Warn().Str("topic", topic).Str("worker", workerName).Msg("Retries exhausted for worker")
	k.decrementWorker(group, topic, workerName) // Decrement the worker count as the worker can no longer process messages.
}

// consumeKafkaTopic connects to Kafka and starts consuming messages for a specific topic with the given group ID.
func (k *kafkaConsumerManager) consumeKafkaTopic(group, topic, workerName string) error {
	// Create a new Kafka reader for the topic, specifying brokers and consumer group details.
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.brokers, // Retrieve Kafka brokers from the environment configuration for connection.
		GroupID: group,     // Assign the consumer group ID for coordinated consumption of messages.
		Topic:   topic,     // Specify the topic from which messages will be consumed.
		Dialer: &kafka.Dialer{
			Timeout:   10 * time.Second, // Set a timeout for Kafka connection attempts.
			KeepAlive: 5 * time.Minute,  // Keep the connection alive for 5 minutes to maintain stability.
		},
		HeartbeatInterval: 3 * time.Second, // Interval for sending heartbeats to Kafka brokers to maintain the connection.
		MaxAttempts:       retryAttempts,   // Set the maximum number of attempts to consume a message before giving up.
	})

	// Ensure the Kafka reader is properly closed when the function exits to free resources.
	defer func() {
		if err := r.Close(); err != nil {
			log.Error().Err(err).Str("worker", workerName).Msg("Failed to close Kafka reader")
		}
	}()

	// Infinite loop to continuously consume messages from the Kafka topic.
	for {
		select {
		case <-k.ctx.Done():
			// If the context is canceled, log the event and stop message consumption gracefully.
			log.Info().Str("worker", workerName).Msg("Context cancelled, shutting down")
			return nil
		default:
			// Fetch the next message from the Kafka topic.
			msg, err := r.FetchMessage(k.ctx)
			if err != nil {
				return err // Return error if message fetching fails, terminating the loop.
			}

			// Process the fetched message using the provided processing function.
			k.ProcessMessage(msg, workerName)

			// Create a 1-minute context for committing the message offset.
			commitCtx, commitCancel := context.WithTimeout(context.Background(), 1*time.Minute)

			// Commit the message offset to Kafka to mark the message as successfully processed.
			if err = r.CommitMessages(commitCtx, msg); err != nil {
				// Log an error if committing the message offset fails.
				logger.LogErrorWithKafkaMessage(err, workerName, msg, "Commit failed for message offset")
			}

			// Call the cancel function to release resources after each message commit.
			commitCancel()
		}
	}
}
