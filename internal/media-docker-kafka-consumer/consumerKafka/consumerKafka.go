package consumerKafka

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
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
		topic = "video-response"
		id, resMessage, err = processVideoMessage(msg.Value)
	case "video-resolutions":
		topic = "video-resolutions-response"
		id, resMessage, err = processVideoResolutionsMessage(msg.Value)
	case "image":
		topic = "image-response"
		id, resMessage, err = processImageMessage(msg.Value)
	case "audio":
		topic = "audio-response"
		id, resMessage, err = processAudioMessage(msg.Value)
	case "delete-file":
		processDeleteFileMessage(msg, workerName) // Process file deletion request
		return
	default:
		// Log unknown topic error
		log.Error().
			Str("worker", workerName).
			Interface("message_details", map[string]interface{}{
				"topic":         msg.Topic,
				"partition":     msg.Partition,
				"offset":        msg.Offset,
				"highWaterMark": msg.HighWaterMark,
				"value":         string(msg.Value),
				"time":          msg.Time,
			}).
			Msg("Unknown topic")
		return
	}

	// Handle errors and prepare response message
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
			Str("worker", workerName).
			Any("new_kafka_message", message).
			Str("response topic", topic).
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
