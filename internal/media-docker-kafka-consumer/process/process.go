// process package contains Kafka message processing functions for media-docker-kafka-consumer.
package process

import (
	"fmt"
	"os"
	"time"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

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
		newId, err = helper.ExtractNewId(msg.Value)
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

	// NOTE: No response will be sent to "media-docker-files-response",
	// leaving client backend services unnotified.
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("dlq_message", dlqMessage).
		Str("newId", "Failed to get newId"). // Log missing ID error.
		Msg("Error producing message to failed-letter-queue.")
}

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

// processVideoMessage processes video conversion and returns the new ID, message, or an error
func processVideoMessage(kafkaMsg []byte) (string, string, error) {
	var videoMsg topics.VideoMessage

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
	var videoResolutionsMsg topics.VideoResolutionsMessage

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
	var imageMsg topics.ImageMessage

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
	var audioMsg topics.AudioMessage // Corrected type from ImageMessage to AudioMessage

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
