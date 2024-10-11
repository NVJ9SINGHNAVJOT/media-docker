package consumerKafka

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
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

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var err error
	var fileType string
	var id string
	var resMessage string

	// Process messages based on their topic
	switch msg.Topic {
	case "video":
		fileType = "video"                                   // Assign file type for video messages
		id, resMessage, err = processVideoMessage(msg.Value) // Process the video message
	case "video-resolutions":
		fileType = "videoResolutions"                                   // Assign file type for video resolution messages
		id, resMessage, err = processVideoResolutionsMessage(msg.Value) // Process the video resolution message
	case "image":
		fileType = "image"                                   // Assign file type for image messages
		id, resMessage, err = processImageMessage(msg.Value) // Process the image message
	case "audio":
		fileType = "audio"                                   // Assign file type for audio messages
		id, resMessage, err = processAudioMessage(msg.Value) // Process the audio message
	case "delete-file":
		processDeleteFileMessage(msg, workerName) // Handle file deletion request
		return
	default:
		// Log an error for unknown topics
		log.Error().
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         msg.Topic,         // Topic of the message
				"partition":     msg.Partition,     // Partition info
				"offset":        msg.Offset,        // Offset of the message
				"highWaterMark": msg.HighWaterMark, // High water mark
				"value":         string(msg.Value), // Message content
				"time":          msg.Time,          // Time when the message was received
			}).
			Msg("Unknown topic") // Log message for unknown topic
		return
	}

	// Create a response message object with ID and FileType
	message := KafkaResponseMessage{
		ID:       id,
		FileType: fileType,
	}

	// Handle errors that may have occurred during message processing
	if err != nil {
		// If ID is empty, log an error and return without processing further
		if id == "" {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Interface("message_details", map[string]interface{}{
					"topic":         msg.Topic,
					"partition":     msg.Partition,
					"offset":        msg.Offset,
					"highWaterMark": msg.HighWaterMark,
					"value":         string(msg.Value),
					"time":          msg.Time,
				}).
				Str("id", "ID not returned from message processing"). // Log missing ID error
				Msg(resMessage)
			return
		}
		// If an error occurred but ID exists, mark the status as "failed"
		message.Status = "failed"
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         msg.Topic,
				"partition":     msg.Partition,
				"offset":        msg.Offset,
				"highWaterMark": msg.HighWaterMark,
				"value":         string(msg.Value),
				"time":          msg.Time,
			}).
			Msg(resMessage) // Log error and response message
	} else {
		// If no error, mark the status as "completed"
		message.Status = "completed"
	}

	// Produce the response message to the "media-docker-response" topic
	err = KafkaProducer.Produce("media-docker-response", message)
	if err != nil {
		// Log error if producing the response message fails
		log.Error().
			Err(err).
			Str("worker", workerName).
			Any("new_kafka_message", message). // Log the new Kafka message content
			Str("response topic", "media-docker-response").
			Msg("Error while producing message for response")
	}
}

// processVideoMessage processes video conversion and returns the new ID, message, or an error
func processVideoMessage(kafkaMsg []byte) (string, string, error) {
	var videoMsg api.VideoMessage

	// Unmarshal and Validate the Kafka message into VideoMessage struct
	if errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &videoMsg); err != nil {
		return "", errMsg + " VideoMessage", err
	}

	defer pkg.AddToFileDeleteChan(videoMsg.FilePath) // Ensure file is scheduled for deletion

	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoMsg.NewId)

	// Create the output directory
	if err := pkg.CreateDir(outputPath); err != nil {
		return videoMsg.NewId, "Error creating output directory", err
	}

	// Prepare the command for video conversion based on the quality
	var cmd *exec.Cmd
	if videoMsg.Quality != nil {
		// Use provided quality
		cmd = pkg.ConvertVideo(videoMsg.FilePath, outputPath, *videoMsg.Quality)
	} else {
		// Use default quality
		cmd = pkg.ConvertVideo(videoMsg.FilePath, outputPath)
	}

	// Execute the command
	if err := cmd.Run(); err != nil {
		pkg.AddToDirDeleteChan(outputPath) // Schedule directory for deletion on error
		return videoMsg.NewId, "Video conversion failed", fmt.Errorf("command: %s, %s", cmd.String(), err)
	}

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
	if errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &videoResolutionsMsg); err != nil {
		return "", errMsg + " VideoResolutionsMessage", err
	}

	defer pkg.AddToFileDeleteChan(videoResolutionsMsg.FilePath) // Ensure file is scheduled for deletion

	// Prepare the output directories for each resolution
	outputPaths := map[string]string{
		"360":  fmt.Sprintf("%s/videos/%s/360", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"480":  fmt.Sprintf("%s/videos/%s/480", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"720":  fmt.Sprintf("%s/videos/%s/720", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"1080": fmt.Sprintf("%s/videos/%s/1080", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
	}

	// Create the output directories
	if err := pkg.CreateDirs([]string{outputPaths["360"], outputPaths["480"], outputPaths["720"], outputPaths["1080"]}); err != nil {
		return videoResolutionsMsg.NewId, "Error creating output directories", err
	}

	// Assume outputPaths is a map with resolution as key and output path as value
	for res, outputPath := range outputPaths {
		cmd := pkg.ConvertVideoResolutions(videoResolutionsMsg.FilePath, outputPath, res)

		// Run the command and check for errors
		if err := cmd.Run(); err != nil {
			cleanUpResolutions(outputPaths)
			return videoResolutionsMsg.NewId, "Video conversion failed for resolution " + res, fmt.Errorf("command: %s, %s", cmd.String(), err)
		}

	}

	// Return success: new ID and success message
	return videoResolutionsMsg.NewId, "Video resolution conversion completed successfully", nil
}

// processImageMessage processes image conversion and returns the new ID, message, or an error
func processImageMessage(kafkaMsg []byte) (string, string, error) {
	var imageMsg api.ImageMessage

	// Unmarshal and Validate the Kafka message into ImageMessage struct
	if errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &imageMsg); err != nil {
		return "", errMsg + " ImageMessage", err
	}

	defer pkg.AddToFileDeleteChan(imageMsg.FilePath) // Ensure file is scheduled for deletion

	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, imageMsg.NewId)

	// Prepare the command for image conversion based on the quality
	cmd := pkg.ConvertImage(imageMsg.FilePath, outputPath, "1")

	// Execute the command
	if err := cmd.Run(); err != nil {
		return imageMsg.NewId, "Image conversion failed", fmt.Errorf("command: %s, %s", cmd.String(), err)
	}

	// Return success: new ID and a success message
	return imageMsg.NewId, "Image conversion completed successfully", nil
}

// processAudioMessage processes audio conversion and returns the new ID, message, or an error
func processAudioMessage(kafkaMsg []byte) (string, string, error) {
	var audioMsg api.AudioMessage // Corrected type from ImageMessage to AudioMessage

	// Unmarshal the Kafka message into AudioMessage struct
	if errMsg, err := helper.UnmarshalAndValidate(kafkaMsg, &audioMsg); err != nil {
		return "", errMsg + " AudioMessage", err
	}

	defer pkg.AddToFileDeleteChan(audioMsg.FilePath) // Ensure file is scheduled for deletion

	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, audioMsg.NewId)

	// Prepare the command for audio conversion using the provided bitrate (if any)
	var cmd *exec.Cmd
	if audioMsg.Bitrate != nil {
		cmd = pkg.ConvertAudio(audioMsg.FilePath, outputPath, *audioMsg.Bitrate)
	} else {
		cmd = pkg.ConvertAudio(audioMsg.FilePath, outputPath) // Call without bitrate
	}

	// Execute the command
	if err := cmd.Run(); err != nil {
		return audioMsg.NewId, "Audio conversion failed", fmt.Errorf("command: %s, %s", cmd.String(), err)
	}

	// Return success: new ID and a success message
	return audioMsg.NewId, "Audio conversion completed successfully", nil
}

func processDeleteFileMessage(msg kafka.Message, workerName string) {
	var deleteFileMsg api.DeleteFileRequest

	// Unmarshal and Validate the Kafka message into DeleteFileRequest struct to retrieve the delete request details
	if errMsg, err := helper.UnmarshalAndValidate(msg.Value, &deleteFileMsg); err != nil {
		// Log an error if unmarshalling fails, including message details for troubleshooting
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         msg.Topic,
				"partition":     msg.Partition,
				"offset":        msg.Offset,
				"highWaterMark": msg.HighWaterMark,
				"value":         string(msg.Value),
				"time":          msg.Time,
			}).
			Msg(errMsg + " DeleteFileMessage")
		return
	}

	// Construct the file path based on the media type and ID
	path := fmt.Sprintf("%s/%ss/%s", helper.Constants.MediaStorage, deleteFileMsg.Type, deleteFileMsg.Id)

	/*
		NOTE: No error is passed in the response, as file or directory deletion
		logging is handled inside the deletion function only.
	*/

	var err error
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
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         msg.Topic,
				"partition":     msg.Partition,
				"offset":        msg.Offset,
				"highWaterMark": msg.HighWaterMark,
				"value":         string(msg.Value),
				"time":          msg.Time,
			}).
			Msgf("Error while deleting %s file, id: %s, path: %s", deleteFileMsg.Type, deleteFileMsg.Id, path)
	}
}
