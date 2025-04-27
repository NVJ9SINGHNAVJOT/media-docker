// process package contains Kafka message processing functions for media-docker-failed-consumer.
package process

import (
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// topicHandler is a struct that holds the fileType and the corresponding processing function for a given topic.
type topicHandler struct {
	fileType    string                                          // Describes the type of file (e.g., "video", "audio").
	processFunc func(string, topics.DLQMessage) (string, error) // Function to process the message for the file type and return newId,error if any.
}

// topicHandlers is a map that associates Kafka topics with their respective handlers (fileType and processing function).
var topicHandlers = map[string]topicHandler{
	"video":             {fileType: "video", processFunc: processVideoMessage},                       // Handler for video topic.
	"video-resolutions": {fileType: "videoResolutions", processFunc: processVideoResolutionsMessage}, // Handler for video-resolutions topic.
	"image":             {fileType: "image", processFunc: processImageMessage},                       // Handler for image topic.
	"audio":             {fileType: "audio", processFunc: processAudioMessage},                       // Handler for audio topic.
}

// ProcessMessage processes the Kafka message based on its topic.
// If the message is successfully unmarshalled and validated, it calls handleDLQMessage.
// If not, it attempts to extract the newId and originalTopic from the message,
// logging the error and sending a failed response if necessary.
func ProcessMessage(msg kafka.Message, workerName string) {
	var dlqMsg topics.DLQMessage

	// Unmarshal and validate the message.
	errmsg, err := validator.UnmarshalAndValidate(msg.Value, &dlqMsg)

	if err == nil {
		// Handle the DLQ message if unmarshalling was successful.
		handleDLQMessage(dlqMsg, workerName)
		return
	}

	// Attempt to extract newId and originalTopic from the message on failure.
	newId, originalTopic, extractErr := validator.ExtractNewIdAndOriginalTopic(msg.Value)
	if extractErr == nil {
		// Check if the original topic exists in the topicHandlers map.
		handler, exists := topicHandlers[originalTopic]
		if exists {
			// Log the error, record the failed message processing, and send a failed response.
			logger.LogErrorWithKafkaMessage(err, workerName, msg, errmsg+" DLQMessage")
			kafkahandler.SendConsumerResponse(workerName, newId, handler.fileType, "failed")
			return
		}
	}

	// Log the error if newId and originalTopic extraction fails, without sending a response.
	// CAUTION: No response will be sent to "media-docker-files-response",
	// leaving client backend services unnotified.
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("dlq_message", dlqMsg).
		Str("failed", "Failed to get newId and originalTopic").
		Msg(errmsg + " DLQMessage")
}

// handleDLQMessage processes a "failed-letter-queue" message by first checking if the original topic is recognized.
// It calls the corresponding processing function for the known topic.
// If the original topic is unknown, an error is logged and no response is sent.
// If the message processing fails, the error is logged, and a failure response is sent back to the consumer.
func handleDLQMessage(dlqMsg topics.DLQMessage, workerName string) {

	// Verify that the originalTopic exists in the topicHandlers map.
	//
	// NOTE: The existence of the handler is not checked because
	// the originalTopic has already been validated in the ProcessMessage function.
	handler := topicHandlers[dlqMsg.OriginalTopic]

	// Log a success message if the DLQ message is recognized with a valid originalTopic.
	log.Info().
		Str("worker", workerName).
		Interface("dlq_message", dlqMsg).
		Msg("DLQMessage received.")

	// Process the DLQ message using the appropriate handler function for the original topic.
	newId, err := handler.processFunc(workerName, dlqMsg)
	if err == nil {
		// Log success after processing the DLQ message without errors.
		log.Info().
			Str("worker", workerName).
			Interface("dlq_message", dlqMsg).
			Msg("DLQMessage processing completed successfully.")
		// Send a success response to the consumer indicating the message processing is completed.
		kafkahandler.SendConsumerResponse(workerName, newId, handler.fileType, "completed")
		return
	}

	// Log an error if the processing of the DLQ message fails.
	// INFO: This indicates that the last attempt at file conversion or processing has failed.
	if newId == "" && (dlqMsg.NewId == nil) {
		// CAUTION: No response will be sent to "media-docker-files-response",
		// which leaves client backend services unnotified.
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMsg).
			Str("newId", "Failed to get newId"). // Log an error indicating that newId could not be obtained.
			Msg("Failed to process DLQMessage.")
		return
	}

	// Update the DLQ message with the newId if it is empty.
	if newId == "" {
		*dlqMsg.NewId = newId
	}

	// Log the error indicating the processing of the DLQ message failed.
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("dlq_message", dlqMsg).
		Msg("Failed to process DLQMessage.")
	// Send a failure response to the consumer indicating that the processing has failed.
	kafkahandler.SendConsumerResponse(workerName, newId, handler.fileType, "failed")
}
