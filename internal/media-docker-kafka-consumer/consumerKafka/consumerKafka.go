package consumerKafka

import (
	"fmt"
	"os"
	"time"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// KafkaResponseMessage represents a message from the Media Docker system.
type KafkaResponseMessage struct {
	ID       string `json:"id" validate:"required,uuid4"`                                          // Unique identifier (UUIDv4) for the media file, required field
	FileType string `json:"fileType" validate:"required,oneof=image video videoResolutions audio"` // Media file type, required and must be one of "image", "video", "videoResolutions", or "audio"
	Status   string `json:"status" validate:"required,oneof=completed failed"`                     // Status of the media processing, required and must be either "completed" or "failed"
}

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager
var KafkaConsumer *ka.KafkaConsumerManager

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

// handleErrorResponse processes errors from message consumption functions.
// If an error occurs, a DLQMessage is sent to the "failed-letter-queue" topic
// for further processing by consumer workers in a different service.
// If sending the DLQ message fails, the error is logged.
// If a newId is provided, the function calls SendConsumerResponse with a "failed" status,
// sending a response message to the "media-docker-files-response" topic.
// If newId is not provided, an error is logged indicating the missing newId.
//
// NOTE: If producing the DLQ message and retrieving the ID both fail,
// no response will be sent to the "media-docker-files-response" topic,
// leaving client backend services unnotified.
func handleErrorResponse(msg kafka.Message, workerName, fileType, newId, resMessage string, err error) {
	logger.LogErrorWithKafkaMessage(err, workerName, msg, resMessage)

	// NOTE: Failed messages are sent to the "failed-letter-queue" topic,
	// enabling further processing and reducing retry load on the main consumption service.
	//
	// Create a DLQMessage struct with error details and original message information.
	dlqMessage := ka.DLQMessage{
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

	// Attempt to produce the DLQ message to the "failed-letter-queue" topic.
	err = KafkaProducer.Produce("failed-letter-queue", dlqMessage)

	if err == nil {
		return
	}

	if newId != "" {
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMessage).
			Msg("Error producing message to failed-letter-queue.")
		SendConsumerResponse(workerName, newId, fileType, "failed")
		return
	}

	// If newId is empty, attempt to extract it from the message value.
	// If still unable to retrieve the ID, log the error.
	newId, iderr := helper.ExtractNewId(msg.Value)
	if iderr != nil {
		// NOTE: No response will be sent to "media-docker-files-response",
		// leaving client backend services unnotified.
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMessage).
			Str("id", "ID not returned from message processing"). // Log missing ID error.
			Msg("Error producing message to failed-letter-queue.")
	} else {
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMessage).
			Msg("Error producing message to failed-letter-queue.")
		SendConsumerResponse(workerName, newId, fileType, "failed")
	}
}

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var err error
	var fileType string
	var newId string
	var resMessage string

	// Process messages based on their topic
	switch msg.Topic {
	case "video":
		fileType = "video"                                      // Assign file type for video messages
		newId, resMessage, err = processVideoMessage(msg.Value) // Process the video message
	case "video-resolutions":
		fileType = "videoResolutions"                                      // Assign file type for video resolution messages
		newId, resMessage, err = processVideoResolutionsMessage(msg.Value) // Process the video resolution message
	case "image":
		fileType = "image"                                      // Assign file type for image messages
		newId, resMessage, err = processImageMessage(msg.Value) // Process the image message
	case "audio":
		fileType = "audio"                                      // Assign file type for audio messages
		newId, resMessage, err = processAudioMessage(msg.Value) // Process the audio message
	case "delete-file":
		processDeleteFileMessage(msg, workerName) // Handle file deletion request
		return
	default:
		// Log an error for unknown topics
		logger.LogUnknownTopic(workerName, msg)
		return
	}

	// Handle errors that may have occurred during message processing
	if err != nil {
		handleErrorResponse(msg, workerName, fileType, newId, resMessage, err)
		return
	}

	// If no errors occurred during processing, send a success response
	SendConsumerResponse(workerName, newId, fileType, "completed")
}

// processVideoMessage processes video conversion and returns the new ID, message, or an error
func processVideoMessage(kafkaMsg []byte) (string, string, error) {
	var videoMsg api.VideoMessage

	// Unmarshal and Validate the Kafka message into VideoMessage struct
	errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &videoMsg)
	if err != nil {
		return "", errMsg + " VideoMessage", err
	}

	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoMsg.NewId)

	// Create the output directory
	if err = pkg.CreateDir(outputPath); err != nil {
		return videoMsg.NewId, "Error creating output directory", err
	}

	// Execute the command for video conversion based on the quality
	if videoMsg.Quality != nil {
		// Use provided quality
		err = pkg.ConvertVideo(videoMsg.FilePath, outputPath, *videoMsg.Quality)
	} else {
		// Use default quality
		err = pkg.ConvertVideo(videoMsg.FilePath, outputPath)
	}
	if err != nil {
		pkg.AddToDirDeleteChan(outputPath) // Schedule directory for deletion on error
		return videoMsg.NewId, "Video conversion failed", err
	}

	pkg.AddToFileDeleteChan(videoMsg.FilePath) // Ensure file is scheduled for deletion

	// Return success: new ID and a success message
	return videoMsg.NewId, "Video conversion completed successfully", nil
}

func cleanUpResolutions(outputPaths map[string]string) {
	for _, outputPath := range outputPaths {
		pkg.AddToDirDeleteChan(outputPath)
	}
}

// processVideoResolutionsMessage processes video resolution conversion and returns the new ID, message, or an error
func processVideoResolutionsMessage(kafkaMsg []byte) (string, string, error) {
	var videoResolutionsMsg api.VideoResolutionsMessage

	// Unmarshal and Validate the Kafka message into VideoResolutionsMessage struct
	errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &videoResolutionsMsg)
	if err != nil {
		return "", errMsg + " VideoResolutionsMessage", err
	}

	// Prepare the output directories for each resolution
	outputPaths := map[string]string{
		"360":  fmt.Sprintf("%s/videos/%s/360", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"480":  fmt.Sprintf("%s/videos/%s/480", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"720":  fmt.Sprintf("%s/videos/%s/720", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"1080": fmt.Sprintf("%s/videos/%s/1080", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
	}

	// Create the output directories
	if err = pkg.CreateDirs([]string{outputPaths["360"], outputPaths["480"], outputPaths["720"], outputPaths["1080"]}); err != nil {
		return videoResolutionsMsg.NewId, "Error creating output directories", err
	}

	// Assume outputPaths is a map with resolution as key and output path as value
	for res, outputPath := range outputPaths {
		// Execute the command and check for errors
		if err = pkg.ConvertVideoResolutions(videoResolutionsMsg.FilePath, outputPath, res); err != nil {
			cleanUpResolutions(outputPaths)
			return videoResolutionsMsg.NewId, "Video conversion failed for resolution " + res, err
		}
	}

	pkg.AddToFileDeleteChan(videoResolutionsMsg.FilePath) // Ensure file is scheduled for deletion

	// Return success: new ID and success message
	return videoResolutionsMsg.NewId, "Video resolution conversion completed successfully", nil
}

// processImageMessage processes image conversion and returns the new ID, message, or an error
func processImageMessage(kafkaMsg []byte) (string, string, error) {
	var imageMsg api.ImageMessage

	// Unmarshal and Validate the Kafka message into ImageMessage struct
	errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &imageMsg)
	if err != nil {
		return "", errMsg + " ImageMessage", err
	}

	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, imageMsg.NewId)

	// Execute the command for image processing
	if err = pkg.ConvertImage(imageMsg.FilePath, outputPath, "1"); err != nil {
		return imageMsg.NewId, "Image conversion failed", err
	}

	pkg.AddToFileDeleteChan(imageMsg.FilePath) // Ensure file is scheduled for deletion

	// Return success: new ID and a success message
	return imageMsg.NewId, "Image conversion completed successfully", nil
}

// processAudioMessage processes audio conversion and returns the new ID, message, or an error
func processAudioMessage(kafkaMsg []byte) (string, string, error) {
	var audioMsg api.AudioMessage // Corrected type from ImageMessage to AudioMessage

	// Unmarshal the Kafka message into AudioMessage struct
	errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &audioMsg)
	if err != nil {
		return "", errMsg + " AudioMessage", err
	}

	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, audioMsg.NewId)

	// Execute the command for audio conversion using the provided bitrate (if any)
	if audioMsg.Bitrate != nil {
		err = pkg.ConvertAudio(audioMsg.FilePath, outputPath, *audioMsg.Bitrate)
	} else {
		err = pkg.ConvertAudio(audioMsg.FilePath, outputPath) // Call without bitrate
	}

	if err != nil {
		return audioMsg.NewId, "Audio conversion failed", err
	}

	pkg.AddToFileDeleteChan(audioMsg.FilePath) // Ensure file is scheduled for deletion

	// Return success: new ID and a success message
	return audioMsg.NewId, "Audio conversion completed successfully", nil
}

func processDeleteFileMessage(msg kafka.Message, workerName string) {
	var deleteFileMsg api.DeleteFileRequest

	// Unmarshal and Validate the Kafka message into DeleteFileRequest struct to retrieve the delete request details
	errMsg, err := helper.UnmarshalAndValidate(msg.Value, &deleteFileMsg)
	if err != nil {
		// Log an error if unmarshalling fails, including message details for troubleshooting
		logger.LogErrorWithKafkaMessage(err, workerName, msg, errMsg+" DeleteFileMessage")
		return
	}

	// Construct the file path based on the media type and ID
	path := fmt.Sprintf("%s/%ss/%s", helper.Constants.MediaStorage, deleteFileMsg.Type, deleteFileMsg.Id)

	/*
		NOTE: No error is passed in the response, as file or directory deletion
		logging is handled inside the deletion function only.
	*/

	// Determine the media type and call the appropriate deletion function
	if deleteFileMsg.Type == "image" {
		err = os.Remove(path + ".jpeg") // Delete the image file with a .jpeg extension
	} else if deleteFileMsg.Type == "audio" {
		err = os.Remove(path + ".mp3") // Delete the audio file with a .mp3 extension
	} else {
		err = os.RemoveAll(path) // Delete the directory for other types
	}

	// Log any error that occurs during deletion, along with relevant details for troubleshooting
	if err != nil {
		logger.LogErrorWithKafkaMessage(
			err,
			workerName,
			msg,
			fmt.Sprintf("Error while deleting %s file, id: %s, path: %s", deleteFileMsg.Type, deleteFileMsg.Id, path))
	}
}
