package consumerKafka

import (
	"encoding/json" // For JSON marshaling and unmarshaling
	"fmt"           // For formatted I/O operations
	"os"
	"os/exec" // For executing external commands

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"     // For structured logging
	"github.com/segmentio/kafka-go" // Kafka client library
)

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager
var KafkaConsumer *ka.KafkaConsumerManager

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var err error
	var topic string
	var id string
	var resMessage string
	var success bool

	// Process messages by topic
	switch msg.Topic {
	case "video":
		topic = "videoResponse"
		id, resMessage, err = processVideoMessage(msg)
	case "videoResolutions":
		topic = "videoResolutionsResponse"
		id, resMessage, err = processVideoResolutionsMessage(msg)
	case "image":
		topic = "imageResponse"
		id, resMessage, err = processImageMessage(msg)
	case "audio":
		topic = "audioResponse"
		id, resMessage, err = processAudioMessage(msg)
	case "deleteFile":
		processDeleteFileMessage(msg, workerName) // Process file deletion request
		return
	default:
		// Log unknown topic error
		log.Error().
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg("Unknown topic")
		return
	}

	// Handle errors and prepare response message
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg(resMessage)
		if id == "" {
			return // Exit if no ID is available
		}
		success = false
	} else {
		success = true
		resMessage = fmt.Sprintf("Successfully processed message for topic: %s", msg.Topic)
	}

	// Create response message
	message := serverKafka.KafkaResponseMessage{
		ID:      id,
		Success: success,
		Message: resMessage,
	}
	// Produce the response message
	err = KafkaProducer.Produce(topic, message)
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", topic).
			Str("worker", workerName).
			Any("kafka_message", message).
			Msg("Error while producing message for response")
	}
}

// processVideoMessage processes video conversion and returns the new ID, message, or an error
func processVideoMessage(kafkaMsg kafka.Message) (string, string, error) {
	var videoMsg api.VideoMessage

	// Unmarshal the Kafka message into VideoMessage struct
	if err := json.Unmarshal(kafkaMsg.Value, &videoMsg); err != nil {
		return "", "Failed to unmarshal VideoMessage", err
	}

	// Validate the unmarshaled struct
	if err := helper.ValidateStruct(videoMsg); err != nil {
		return "", "Validation failed for VideoMessage", err
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
		return videoMsg.NewId, "Video conversion failed", err
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
func processVideoResolutionsMessage(kafkaMsg kafka.Message) (string, string, error) {
	var videoResolutionsMsg api.VideoResolutionsMessage

	// Unmarshal the Kafka message into VideoResolutionsMessage struct
	if err := json.Unmarshal(kafkaMsg.Value, &videoResolutionsMsg); err != nil {
		return "", "Failed to unmarshal VideoResolutionsMessage", err
	}

	// Validate the unmarshaled struct
	if err := helper.ValidateStruct(videoResolutionsMsg); err != nil {
		return "", "Validation failed for VideoResolutionsMessage", err
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
			return videoResolutionsMsg.NewId, "Video conversion failed for resolution " + res, err
		}

	}

	// Return success: new ID and success message
	return videoResolutionsMsg.NewId, "Video resolution conversion completed successfully", nil
}

// processImageMessage processes image conversion and returns the new ID, message, or an error
func processImageMessage(kafkaMsg kafka.Message) (string, string, error) {
	var imageMsg api.ImageMessage

	// Unmarshal the Kafka message into ImageMessage struct
	if err := json.Unmarshal(kafkaMsg.Value, &imageMsg); err != nil {
		return "", "Failed to unmarshal ImageMessage", err
	}

	// Validate the unmarshaled struct
	if err := helper.ValidateStruct(imageMsg); err != nil {
		return "", "Validation failed for ImageMessage", err
	}
	defer pkg.AddToFileDeleteChan(imageMsg.FilePath) // Ensure file is scheduled for deletion

	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, imageMsg.NewId)

	// Prepare the command for image conversion based on the quality
	cmd := pkg.ConvertImage(imageMsg.FilePath, outputPath, "1")

	// Execute the command
	if err := cmd.Run(); err != nil {
		return imageMsg.NewId, "Image conversion failed", err
	}

	// Return success: new ID and a success message
	return imageMsg.NewId, "Image conversion completed successfully", nil
}

// processAudioMessage processes audio conversion and returns the new ID, message, or an error
func processAudioMessage(kafkaMsg kafka.Message) (string, string, error) {
	var audioMsg api.AudioMessage // Corrected type from ImageMessage to AudioMessage

	// Unmarshal the Kafka message into AudioMessage struct
	if err := json.Unmarshal(kafkaMsg.Value, &audioMsg); err != nil {
		return "", "Failed to unmarshal AudioMessage", err
	}

	// Validate the unmarshaled struct
	if err := helper.ValidateStruct(audioMsg); err != nil {
		return "", "Validation failed for AudioMessage", err
	}
	defer pkg.AddToFileDeleteChan(audioMsg.FilePath) // Ensure file is scheduled for deletion

	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, audioMsg.NewId)

	// Prepare the command for audio conversion
	cmd := pkg.ConvertAudio(audioMsg.FilePath, outputPath)

	// Execute the command
	if err := cmd.Run(); err != nil {
		return audioMsg.NewId, "Audio conversion failed", err
	}

	// Return success: new ID and a success message
	return audioMsg.NewId, "Audio conversion completed successfully", nil
}

func processDeleteFileMessage(kafkaMsg kafka.Message, workerName string) {
	var deleteFileMsg api.DeleteFileRequest

	// Unmarshal the Kafka message into DeleteFileRequest struct to retrieve the delete request details
	if err := json.Unmarshal(kafkaMsg.Value, &deleteFileMsg); err != nil {
		// Log an error if unmarshalling fails, including message details for troubleshooting
		log.Error().
			Err(err).
			Str("topic", kafkaMsg.Topic).
			Int("partition", kafkaMsg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(kafkaMsg.Value)).
			Msg("Failed to unmarshal DeleteFileMessage")
		return
	}

	// Validate the unmarshaled struct to ensure it contains the necessary information
	if err := helper.ValidateStruct(deleteFileMsg); err != nil {
		// Log an error if validation fails, including message details for troubleshooting
		log.Error().
			Err(err).
			Str("topic", kafkaMsg.Topic).
			Int("partition", kafkaMsg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(kafkaMsg.Value)).
			Msg("Validation failed for DeleteFileMessage")
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
			Str("topic", kafkaMsg.Topic).
			Int("partition", kafkaMsg.Partition).
			Str("worker", workerName).
			Interface("kafka_message", deleteFileMsg).
			Msgf("Error while deleting %s file, id: %s", deleteFileMsg.Type, deleteFileMsg.Id)
	}
}
