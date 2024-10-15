package mediadockerkafka

import "github.com/rs/zerolog/log"

// KafkaResponseMessage represents a message from the Media Docker system.
type KafkaResponseMessage struct {
	ID       string `json:"id" validate:"required,uuid4"`                                          // Unique identifier (UUIDv4) for the media file, required field
	FileType string `json:"fileType" validate:"required,oneof=image video videoResolutions audio"` // Media file type, required and must be one of "image", "video", "videoResolutions", or "audio"
	Status   string `json:"status" validate:"required,oneof=completed failed"`                     // Status of the media processing, required and must be either "completed" or "failed"
}

// SendConsumerResponse produces a Kafka message to the "media-docker-files-response" topic.
//
// Parameters:
// - workerName: Name of the worker processing the message.
// - newId: Unique identifier for the message being processed.
// - fileType: Type of the file being processed. Allowed values:
//   - "video"
//   - "videoResolutions"
//   - "image"
//   - "audio"
//
// - status: Status of the file processing. Allowed values:
//   - "completed"
//   - "failed"
//
// NOTE: Providing values outside the allowed range for fileType or status may cause
// errors during further processing by client backend services.
func SendConsumerResponse(workerName, newId, fileType, status string) {
	// Create a response message object with the provided ID, FileType, and Status.
	message := KafkaResponseMessage{
		ID:       newId,
		FileType: fileType,
		Status:   status,
	}

	// Produce the response message to the "media-docker-files-response" topic.
	err := KafkaProducer.Produce("media-docker-files-response", message)
	if err != nil {
		// Log an error if producing the response message fails.
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("new_kafka_message", message). // Log the new Kafka message content.
			Str("response_topic", "media-docker-files-response").
			Msg("Error while producing message for response.")
	}
}
