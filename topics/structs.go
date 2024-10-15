// package topics contains structs for all Kafka topics used in Media Docker.
// Each Kafka topic's message value contains data based on the structs defined here.
// These structs are also used for validation.
//
// NOTE: Any updates to the structs must be done carefully, as after changes,
// previous messages for a particular topic will need to be handled with the
// previous struct version to ensure compatibility.
package topics

import "time"

// INFO: All topics are listed in the root folder in `./kafka_config.sh`.

// DLQMessage represents the structure of messages sent to the "failed-letter-queue",
// acting as the Dead-Letter Queue (DLQ) for this project.
// It stores metadata about the original message, the error encountered, and additional processing details.
//
// Topic: "failed-letter-queue"
type DLQMessage struct {
	OriginalTopic  string    `json:"originalTopic" validate:"required"`  // The topic where the message originated
	Partition      int       `json:"partition" validate:"required"`      // Kafka partition of the original message
	Offset         int64     `json:"offset" validate:"required"`         // Offset position of the original message in the partition
	HighWaterMark  int64     `json:"highWaterMark" validate:"required"`  // The high-water mark of the partition (latest offset)
	Value          string    `json:"value" validate:"required"`          // The original message content as a string
	ErrorDetails   string    `json:"errorDetails" validate:"required"`   // Description of the error encountered during processing
	ProcessingTime time.Time `json:"processingTime" validate:"required"` // Timestamp of when the message was processed
	ErrorTime      time.Time `json:"errorTime" validate:"required"`      // Timestamp of when the error occurred
	Worker         string    `json:"worker" validate:"required"`         // Identifier of the worker that processed the message
	CustomMessage  string    `json:"customMessage" validate:"required"`  // Additional custom message or context about the error
}

// KafkaResponseMessage represents a message from the Media Docker system.
//
// Topic: "media-docker-files-response"
type KafkaResponseMessage struct {
	ID       string `json:"id" validate:"required,uuid4"`                                          // Unique identifier (UUIDv4) for the media file, required field
	FileType string `json:"fileType" validate:"required,oneof=image video videoResolutions audio"` // Media file type, required and must be one of "image", "video", "videoResolutions", or "audio"
	Status   string `json:"status" validate:"required,oneof=completed failed"`                     // Status of the media processing, required and must be either "completed" or "failed"
}

// AudioMessage represents the structure of the message sent to Kafka for audio processing.
//
// Topic: "audio"
type AudioMessage struct {
	FilePath string  `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string  `json:"newId" validate:"required"`    // New unique identifier for the audio file URL
	Bitrate  *string `json:"bitrate" validate:"omitempty"` // Optional quality parameter
}

// ImageMessage represents the structure of the message sent to Kafka for image processing.
//
// Topic: "image"
type ImageMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New unique identifier for the image file URL
}

// VideoMessage represents the structure of the message sent to Kafka for video processing.
//
// Topic: "video"
type VideoMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New unique identifier for the video file URL
	Quality  *int   `json:"quality" validate:"omitempty"` // Optional video quality (using pointer for omitempty)
}

// VideoResolutionsMessage represents the structure of the message sent to Kafka for video resolution processing.
//
// Topic: "video-resolutions"
type VideoResolutionsMessage struct {
	FilePath string `json:"filePath" validate:"required"` // Mandatory field for the file path
	NewId    string `json:"newId" validate:"required"`    // New unique identifier for the video file URL
}
