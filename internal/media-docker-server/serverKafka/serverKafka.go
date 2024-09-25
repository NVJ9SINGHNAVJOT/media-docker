package serverKafka

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nvj9singhnavjot/media-docker/helper"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Global Maps for Request to Kafka Consumer
var (
	VideoRequestMap           sync.Map
	VideoResolutionRequestMap sync.Map
	ImageRequestMap           sync.Map
	AudioRequestMap           sync.Map
)

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager
var KafkaConsumer *ka.KafkaConsumerManager

// Define the structure for the incoming Kafka messages
type KafkaResponseMessage struct {
	ID      string `json:"id" validate:"required,uuid4"` // Unique ID for identifying the request
	Success bool   `json:"success" validate:"required"`  // Whether the request was successful
	Message string `json:"message" validate:"required"`  // Additional message data
}

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var response KafkaResponseMessage

	// Unmarshal the incoming message into kafkaResponseMessage struct
	err := json.Unmarshal(msg.Value, &response)
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg("Failed to unmarshal Kafka message")
		return
	}

	// Validate the unmarshaled message data
	if err = helper.ValidateStruct(response); err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg("Validation failed for KafkaResponseMessage")
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
			Str("kafka_message", string(msg.Value))
	}
}

// handleResponse retrieves the channel from the map using the message ID and sends the success value
func handleResponse(response KafkaResponseMessage, topic string) error {
	// Select the correct map based on the topic
	var requestMap *sync.Map
	switch topic {
	case "videoResponse":
		requestMap = &VideoRequestMap
	case "videoResolutionResponse":
		requestMap = &VideoResolutionRequestMap
	case "imageResponse":
		requestMap = &ImageRequestMap
	case "audioResponse":
		requestMap = &AudioRequestMap
	default:
		return fmt.Errorf("unknown topic")
	}

	// Retrieve the channel from the selected map using the response ID
	channelValue, ok := requestMap.Load(response.ID)
	if !ok {
		return fmt.Errorf("channel not found in map for given ID")
	}

	// Send the success value to the channel
	channelValue.(chan bool) <- response.Success // Directly using the bool channel type
	return nil
}
