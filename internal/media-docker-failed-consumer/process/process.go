// process package contains Kafka message processing functions for media-docker-failed-consumer.
package process

import (
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// topicHandler is a struct that holds the fileType and the corresponding processing function for a given topic.
type topicHandler struct {
	fileType    string                        // Describes the type of file (e.g., "video", "audio").
	processFunc func(topics.DLQMessage) error // Function to process the message for the file type and return error if any.
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
	errmsg, err := helper.UnmarshalAndValidate(msg.Value, &dlqMsg)

	if err == nil {
		// Handle the DLQ message if unmarshalling was successful.
		handleDLQMessage(dlqMsg, workerName)
		return
	}

	// Attempt to extract newId and originalTopic from the message on failure.
	newId, originalTopic, extractErr := helper.ExtractNewIdAndOriginalTopic(msg.Value)
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
	// NOTE: No response will be sent to "media-docker-files-response",
	// leaving client backend services unnotified.
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("dlq_message", dlqMsg).
		Str("failed", "Failed to get newId and originalTopic").
		Msg(errmsg + " DLQMessage")
}

// handleDLQMessage processes the DLQ message by checking if the originalTopic is known,
// then calls the corresponding processing function for that topic.
// If the originalTopic is not found, an error is logged and no response is sent.
// If the message processing fails, it logs the error and sends a failure response.
func handleDLQMessage(dlqMsg topics.DLQMessage, workerName string) {

	// Check if the originalTopic exists in the topicHandlers map.
	handler, exists := topicHandlers[dlqMsg.OriginalTopic]
	if !exists {
		// Log error for unknown original topic in the DLQ message.
		// NOTE: No response will be sent to "media-docker-files-response",
		// leaving client backend services unnotified about this failure.
		log.Error().
			Str("worker", workerName).
			Interface("dlq_message", dlqMsg).
			Msg("Unknown originalTopic in DLQMessage")
		return
	} else {
		// Log success message if the DLQ message is recognized with a valid originalTopic.
		log.Info().
			Str("worker", workerName).
			Interface("dlq_message", dlqMsg).
			Msg("DLQMessage received.")
	}

	// Process the DLQ message using the appropriate handler function.
	err := handler.processFunc(dlqMsg)
	if err != nil {
		// Log error if the processing of the DLQ message fails.
		// INFO: This indicates the last attempt for file conversion or processing failed.
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMsg).
			Msg("Failed to process DLQMessage.")
		// Send a failure response to the consumer indicating the processing has failed.
		kafkahandler.SendConsumerResponse(workerName, *dlqMsg.NewId, handler.fileType, "failed")
		return
	}

	// Log success after successfully processing the DLQ message.
	log.Info().
		Str("worker", workerName).
		Interface("dlq_message", dlqMsg).
		Msg("DLQMessage message processing completed successfully.")
	// Send a success response to the consumer indicating the message processing is completed.
	kafkahandler.SendConsumerResponse(workerName, *dlqMsg.NewId, handler.fileType, "completed")
}

func processVideoMessage(dlqMsg topics.DLQMessage) error {

}
func processVideoResolutionsMessage(dlqMsg topics.DLQMessage) error {

}
func processImageMessage(dlqMsg topics.DLQMessage) error {

}
func processAudioMessage(dlqMsg topics.DLQMessage) error {

}
