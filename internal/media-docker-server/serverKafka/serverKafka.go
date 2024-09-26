package serverKafka

import (
	"encoding/json" // Import for JSON encoding/decoding
	"fmt"           // Import for formatting strings
	"sync"          // Import for synchronization primitives

	"github.com/nvj9singhnavjot/media-docker/helper"   // Importing helper functions for validation
	ka "github.com/nvj9singhnavjot/media-docker/kafka" // Importing Kafka related functionalities
	"github.com/rs/zerolog/log"                        // Importing zerolog for structured logging
	"github.com/segmentio/kafka-go"                    // Importing kafka-go for Kafka message handling
)

// Global Maps for Request to Kafka Consumer
var (
	VideoRequestMap           sync.Map // Map for tracking video requests
	VideoResolutionRequestMap sync.Map // Map for tracking video resolution requests
	ImageRequestMap           sync.Map // Map for tracking image requests
	AudioRequestMap           sync.Map // Map for tracking audio requests
)

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager // Kafka producer manager for sending messages
var KafkaConsumer *ka.KafkaConsumerManager // Kafka consumer manager for receiving messages

// KafkaResponseMessage defines the structure for the incoming Kafka messages
type KafkaResponseMessage struct {
	ID      string `json:"id" validate:"required,uuid4"` // Unique ID for identifying the request
	Success bool   `json:"success" validate:"required"`  // Indicates whether the request was successful
	Message string `json:"message" validate:"required"`  // Additional message data
}

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var response KafkaResponseMessage

	// Unmarshal the incoming message into KafkaResponseMessage struct
	err := json.Unmarshal(msg.Value, &response)
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg("Failed to unmarshal Kafka message") // Log the error if unmarshalling fails
		return
	}

	// Validate the unmarshalled message data
	if err = helper.ValidateStruct(response); err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg("Validation failed for KafkaResponseMessage") // Log validation failure
		return
	}

	// Handle messages using the ID for channel lookup in the map
	err = handleResponse(response, msg.Topic)

	// If an error occurred, log the error along with context details
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)) // Log error details if handling fails
	}
}

// handleResponse retrieves the channel from the map using the message ID and sends the success value
func handleResponse(response KafkaResponseMessage, topic string) error {
	// Select the correct map based on the topic
	var requestMap *sync.Map
	switch topic {
	case "videoResponse":
		requestMap = &VideoRequestMap // Use the video request map
	case "videoResolutionResponse":
		requestMap = &VideoResolutionRequestMap // Use the video resolution request map
	case "imageResponse":
		requestMap = &ImageRequestMap // Use the image request map
	case "audioResponse":
		requestMap = &AudioRequestMap // Use the audio request map
	default:
		return fmt.Errorf("unknown topic") // Return error if topic is not recognized
	}

	// Retrieve the channel from the selected map using the response ID
	channelValue, ok := requestMap.Load(response.ID) // Load the channel associated with the ID
	if !ok {
		return fmt.Errorf("channel not found in map for given ID") // Return error if channel not found
	}

	// Send the success value to the channel
	channelValue.(chan bool) <- response.Success // Directly using the bool channel type
	return nil
}
