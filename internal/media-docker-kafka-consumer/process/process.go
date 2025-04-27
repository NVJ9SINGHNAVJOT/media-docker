// process package contains Kafka message processing functions for media-docker-kafka-consumer.
package process

import (
	"time"

	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// topicHandler is a struct that holds the fileType and the corresponding processing function for a given topic.
type topicHandler struct {
	fileType    string                               // Describes the type of file (e.g., "video", "audio").
	processFunc func([]byte) (string, string, error) // Function to process the message and return newId, resultMessage, and error.
}

// topicHandlers is a map that associates Kafka topics with their respective handlers (fileType and processing function).
var topicHandlers = map[string]topicHandler{
	"video": {
		fileType:    "video",             // File type for video messages.
		processFunc: processVideoMessage, // Function to process video messages.
	},
	"video-resolutions": {
		fileType:    "videoResolutions",             // File type for video resolution messages.
		processFunc: processVideoResolutionsMessage, // Function to process video resolution messages.
	},
	"image": {
		fileType:    "image",             // File type for image messages.
		processFunc: processImageMessage, // Function to process image messages.
	},
	"audio": {
		fileType:    "audio",             // File type for audio messages.
		processFunc: processAudioMessage, // Function to process audio messages.
	},
}

// handleErrorResponse processes errors from message consumption functions.
// If an error occurs, a DLQMessage is sent to the "failed-letter-queue" topic
// for further processing by consumer workers in a different service.
// If sending the DLQ message fails, the error is logged.
// If a newId is provided, the function calls SendConsumerResponse with a "failed" status,
// sending a response message to the "media-docker-files-response" topic.
// If newId is not provided, an error is logged indicating the missing newId.
//
// CAUTION: If producing the DLQ message and retrieving the ID both fail,
// no response will be sent to the "media-docker-files-response" topic,
// leaving client backend services unnotified.
func handleErrorResponse(msg kafka.Message, workerName, fileType, newId, resMessage string, err error) {
	logger.LogErrorWithKafkaMessage(err, workerName, msg, resMessage)

	// NOTE: Failed messages are sent to the "failed-letter-queue" topic,
	// enabling further processing and reducing retry load on the main consumption service.
	//
	// Create a DLQMessage struct with error details and original message information.
	dlqMessage := topics.DLQMessage{
		OriginalTopic:  msg.Topic,
		Partition:      msg.Partition,
		Offset:         msg.Offset,
		HighWaterMark:  msg.HighWaterMark,
		Value:          string(msg.Value),
		ErrorDetails:   err.Error(),
		ProcessingTime: msg.Time,
		ErrorTime:      time.Now(),
		Worker:         workerName,
		CustomMessage:  resMessage,
	}

	// If newId is empty, attempt to extract it from the message value.
	// If still unable to retrieve the ID, log the error.
	if newId == "" {
		newId, err = validator.ExtractNewId(msg.Value)
		if err == nil {
			dlqMessage.NewId = &newId
		}
	}

	// Attempt to produce the DLQ message to the "failed-letter-queue" topic.
	err = kafkahandler.KafkaProducer.Produce("failed-letter-queue", dlqMessage)

	if err == nil {
		return
	}

	if newId != "" {
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMessage).
			Msg("Error producing message to failed-letter-queue.")
		kafkahandler.SendConsumerResponse(workerName, newId, fileType, "failed")
		return
	}

	// CAUTION: No response will be sent to "media-docker-files-response",
	// leaving client backend services unnotified.
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("dlq_message", dlqMessage).
		Str("newId", "Failed to get newId"). // Log missing ID error.
		Msg("Error producing message to failed-letter-queue.")
}

// ProcessMessage processes Kafka messages based on the topic and the associated handler.
// It sends the appropriate success or error response after processing.
func ProcessMessage(msg kafka.Message, workerName string) {
	var err error
	var newId string
	var resMessage string

	// Lookup the handler for the topic in the map
	handler, found := topicHandlers[msg.Topic]
	if !found {
		// Special case for the "delete-file" topic
		if msg.Topic == "delete-file" {
			processDeleteFileMessage(msg, workerName) // Handle file deletion request.
		} else {
			// Log an error if the topic is unknown
			logger.LogUnknownTopic(workerName, msg)
		}
		return
	}

	// Call the processing function for the specific topic and get the results
	newId, resMessage, err = handler.processFunc(msg.Value)

	// If an error occurred during processing, handle it appropriately
	if err != nil {
		handleErrorResponse(msg, workerName, handler.fileType, newId, resMessage, err)
		return
	}

	// If the message is processed successfully, send a success response
	kafkahandler.SendConsumerResponse(workerName, newId, handler.fileType, "completed")
}
