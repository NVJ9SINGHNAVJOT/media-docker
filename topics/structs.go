// package topics contains structs for all Kafka topics used in Media Docker.
// Each Kafka topic's message value contains data based on the structs defined here.
// These structs are also used for validation.
//
// NOTE: Any updates to the structs must be done carefully, as after changes,
// previous messages for a particular topic will need to be handled with the
// previous struct version to ensure compatibility.
package topics

import "time"

// Topic names. These must match the entries in `./kafka_config.sh`, which is
// what actually creates them in the cluster.
const (
	Video             = "video"
	VideoResolutions  = "video-resolutions"
	Image             = "image"
	Audio             = "audio"
	DeleteFile        = "delete-file"
	FailedLetterQueue = "failed-letter-queue"
)

// All returns every topic the system uses.
func All() []string {
	return []string{Video, VideoResolutions, Image, Audio, DeleteFile, FailedLetterQueue}
}

// INFO: All topics are listed in the root folder in `./kafka_config.sh`.
//
// NOTE: Since v4 the FilePath of a job message points at the raw upload inside
// the publicly served media storage (media_docker_files/<type>s/<id>/original.<ext>),
// not at a private staging directory. That file is already reachable over HTTP
// when the job is produced, and it must not be deleted unless conversion
// succeeds -- deleting it would break a URL a caller is already using.

// DLQMessage represents the structure of messages sent to the "failed-letter-queue",
// acting as the Dead-Letter Queue (DLQ) for this project.
// It stores metadata about the original message, the error encountered, and additional processing details.
//
// Topic: "failed-letter-queue"
type DLQMessage struct {
	NewId          *string   `json:"newId" validate:"omitempty,uuid4"`                                            // Optional NewId from other topic Kafka message
	OriginalTopic  string    `json:"originalTopic" validate:"required,oneof=image video video-resolutions audio"` // The topic where the message originated
	Partition      int       `json:"partition" validate:"customNonNegativeInt"`                                   // Kafka partition of the original message
	Offset         int64     `json:"offset" validate:"customNonNegativeInt"`                                      // Offset position of the original message in the partition
	HighWaterMark  int64     `json:"highWaterMark" validate:"customNonNegativeInt"`                               // The high-water mark of the partition (latest offset)
	Value          string    `json:"value" validate:"required"`                                                   // The original message content as a string
	ErrorDetails   string    `json:"errorDetails" validate:"required"`                                            // Description of the error encountered during processing
	ProcessingTime time.Time `json:"processingTime" validate:"required"`                                          // Timestamp of when the message was processed
	ErrorTime      time.Time `json:"errorTime" validate:"required"`                                               // Timestamp of when the error occurred
	Worker         string    `json:"worker" validate:"required"`                                                  // Identifier of the worker that processed the message
	CustomMessage  string    `json:"customMessage" validate:"required"`                                           // Additional custom message or context about the error
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
	// Compression is the ffmpeg -q:v value, 1 (highest quality) to 31 (lowest).
	// Optional; the consumer applies its default when omitted.
	Compression *int `json:"compression" validate:"omitempty,min=1,max=31"`
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
