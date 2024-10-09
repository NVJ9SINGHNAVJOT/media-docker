package serverKafka

import (
	"fmt"
	"sync"

	"github.com/nvj9singhnavjot/media-docker/helper"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Global Maps for Request to Kafka Consumer
var (
	VideoRequestMap            sync.Map // Map for tracking video requests
	VideoResolutionsRequestMap sync.Map // Map for tracking video resolution requests
	ImageRequestMap            sync.Map // Map for tracking image requests
	AudioRequestMap            sync.Map // Map for tracking audio requests
)

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager // Kafka producer manager for sending messages
var KafkaConsumer *ka.KafkaConsumerManager // Kafka consumer manager for receiving messages

// KafkaResponseMessage defines the structure for the incoming Kafka messages
type KafkaResponseMessage struct {
	ID      string `json:"id" validate:"required,uuid4"` // Unique ID for identifying the request
	Success bool   `json:"success"`                      // Indicates whether the request was successful
	Message string `json:"message" validate:"required"`  // Additional message data
}

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(kafkaMsg kafka.Message, workerName string) {
	var response KafkaResponseMessage

	// Unmarshal the incoming message into KafkaResponseMessage struct
	msg, err := helper.UnmarshalAndValidate(kafkaMsg.Value, &response)
	if err != nil {
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         kafkaMsg.Topic,
				"partition":     kafkaMsg.Partition,
				"offset":        kafkaMsg.Offset,
				"highWaterMark": kafkaMsg.HighWaterMark,
				"value":         string(kafkaMsg.Value),
				"time":          kafkaMsg.Time,
			}).
			Msg(msg + "Kafka message") // Log the error if unmarshalling fails
		return
	}

	// Handle messages using the ID for channel lookup in the map
	err = handleResponse(response, kafkaMsg.Topic)

	// If an error occurred, log the error along with context details
	if err != nil {
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         kafkaMsg.Topic,
				"partition":     kafkaMsg.Partition,
				"offset":        kafkaMsg.Offset,
				"highWaterMark": kafkaMsg.HighWaterMark,
				"value":         string(kafkaMsg.Value),
				"time":          kafkaMsg.Time,
			}).
			Msg("Failed to notify API due to response message error")
	}
}

// handleResponse retrieves the channel from the map using the message ID and sends the success value
func handleResponse(response KafkaResponseMessage, topic string) error {
	// Select the correct map based on the topic
	var requestMap *sync.Map
	switch topic {
	case "video-response":
		requestMap = &VideoRequestMap // Use the video request map
	case "video-resolutions-response":
		requestMap = &VideoResolutionsRequestMap // Use the video resolution request map
	case "image-response":
		requestMap = &ImageRequestMap // Use the image request map
	case "audio-response":
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
