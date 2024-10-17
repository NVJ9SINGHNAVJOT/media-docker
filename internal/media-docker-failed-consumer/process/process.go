// process package contains Kafka message processing functions for media-docker-failed-consumer.
package process

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// topicHandler is a struct that holds the fileType and the corresponding processing function for a given topic.
type topicHandler struct {
	fileType    string                                // Describes the type of file (e.g., "video", "audio").
	processFunc func(string, topics.DLQMessage) error // Function to process the message for the file type and return error if any.
}

// topicHandlers is a map that associates Kafka topics with their respective handlers (fileType and processing function).
var topicHandlers = map[string]topicHandler{
	"video":             {fileType: "video", processFunc: processVideoMessage},                       // Handler for video topic.
	"video-resolutions": {fileType: "videoResolutions", processFunc: processVideoResolutionsMessage}, // Handler for video-resolutions topic.
	"image":             {fileType: "image", processFunc: processImageMessage},                       // Handler for image topic.
	"audio":             {fileType: "audio", processFunc: processAudioMessage},                       // Handler for audio topic.
}

// removeFile attempts to remove the file at the given path.
// It will make up to 3 attempts to delete the file, logging warnings
// on failure and retrying after a 2-second delay between attempts.
// If the file cannot be removed after 3 attempts, it logs an error.
func removeFile(workerName, path string) {
	for i := 1; i <= 3; i++ {
		// Attempt to remove the file at the specified path.
		err := os.Remove(path)
		if err == nil {
			// If the file is successfully deleted, return immediately.
			return
		}

		// If the attempt failed, check if more retries are available.
		if i < 3 {
			// Log a warning with the worker name, file path, and retry count.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d to delete file %s failed. Retrying in 2 seconds...", i, path)

			// Wait for 2 seconds before retrying.
			time.Sleep(2 * time.Second)
		} else {
			// If all retries are exhausted, log an error with the details.
			log.Error().
				Err(err).
				Str("worker", workerName).
				Str("filePath", path).
				Msg("Failed to delete file after 3 attempts")
		}
	}
}

// RemoveDir attempts to remove the specified directory.
// It makes up to 3 attempts to delete the directory, logging warnings on failure
// and retrying after a 2-second delay between attempts. If the directory cannot
// be removed after 3 attempts, it logs an error.
func RemoveDir(workerName, outputPath string) {
	for i := 1; i <= 3; i++ {
		// Attempt to remove the directory at the specified path.
		err := os.Remove(outputPath)
		if err == nil {
			// If the directory is successfully deleted, return immediately.
			return
		}

		// If the attempt failed, check if more retries are available.
		if i < 3 {
			// Log a warning with the worker name, directory path, and retry count.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d to delete directory %s failed. Retrying in 2 seconds...", i, outputPath)

			// Wait for 2 seconds before retrying.
			time.Sleep(2 * time.Second)
		} else {
			// If all retries are exhausted, log an error with the details.
			log.Error().
				Err(err).
				Str("worker", workerName).
				Str("dirPath", outputPath).
				Msg("Failed to delete directory after 3 attempts")
		}
	}
}

// createOutputDirectory attempts to create the specified directory with a maximum of 3 retries.
// It logs an error if all attempts fail and returns the error.
func createOutputDirectory(workerName, outputPath string) error {
	// Retry the directory creation process for a maximum of 3 attempts.
	for attempt := 1; attempt <= 3; attempt++ {
		// Attempt to create the directory.
		err := pkg.CreateDir(outputPath)
		if err == nil {
			// If successful, return immediately.
			return nil
		}

		// If the last attempt fails, log the error and return it.
		if attempt == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Failed to create output directory %s after 3 attempts.", outputPath)
			return fmt.Errorf("failed to create directory after 3 attempts: %s, %v", outputPath, err)
		}

		// Log a warning and retry if the directory creation fails before the final attempt.
		log.Warn().
			Err(err).
			Str("worker", workerName).
			Msgf("Attempt %d to create output directory %s failed. Retrying in 1 second...", attempt, outputPath)

		// Wait for 1 second before retrying.
		time.Sleep(1 * time.Second)
	}

	// This point will not be reached, since the function either returns success or an error after 3 attempts.
	return nil
}

// cleanupOutputDirectory attempts to remove all files and subdirectories within the specified outputPath.
// It does not remove the outputPath itself. The cleanup process is retried up to 3 times in case of failure.
func cleanupOutputDirectory(workerName, outputPath string) error {
	// Retry the cleanup process for a maximum of 3 attempts.
	for attempt := 1; attempt <= 3; attempt++ {
		// Walk through the directory to delete files and subdirectories.
		err := filepath.Walk(outputPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err // Return immediately if an error occurs while walking the directory.
			}

			// Skip the outputPath directory itself but continue cleaning its contents.
			if path == outputPath {
				return nil // Continue walking without deleting the root outputPath.
			}

			// Attempt to remove the file or directory.
			if info.IsDir() {
				// Remove the directory and its contents.
				err = os.RemoveAll(path)
				if err != nil {
					return fmt.Errorf("failed to remove directory %s: %v", path, err)
				}
			} else {
				// Remove the file.
				err = os.Remove(path)
				if err != nil {
					return fmt.Errorf("failed to remove file %s: %v", path, err)
				}
			}
			return nil
		})

		if err == nil {
			// If no error occurred, the cleanup was successful. Return immediately.
			return nil
		}

		// Log a warning and retry if an error occurs, unless it's the final attempt.
		if attempt == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Failed to clean up output directory %s after 3 attempts.", outputPath)
			return fmt.Errorf("failed to clean up output directory after 3 attempts: %s, %v", outputPath, err)
		}

		// Log a warning and wait for 1 second before retrying.
		log.Warn().
			Err(err).
			Str("worker", workerName).
			Msgf("Attempt %d to clean up output directory %s failed. Retrying in 1 second...", attempt, outputPath)

		time.Sleep(1 * time.Second)
	}

	// This point will not be reached, since the function either returns success or an error after 3 attempts.
	return nil
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
	err := handler.processFunc(workerName, dlqMsg)
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

// processVideoMessage processes a video message retrieved from the Dead Letter Queue (DLQ).
// It validates the message, checks for the existence of the input file, manages the output directory,
// and attempts to convert the video file up to three times, implementing error handling and retry logic as necessary.
func processVideoMessage(workerName string, dlqMsg topics.DLQMessage) error {
	var videoMsg topics.VideoMessage

	// Unmarshal and validate the Kafka message into the VideoMessage struct.
	errMsg, err := helper.UnmarshalAndValidate([]byte(dlqMsg.Value), &videoMsg)
	if err != nil {
		return fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
	}

	// Define the output path where the converted video will be stored.
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoMsg.NewId)

	// Check if the output directory already exists.
	if _, err = os.Stat(outputPath); !os.IsNotExist(err) {
		// If it exists, clean up any existing files or subdirectories within it.
		if err = cleanupOutputDirectory(workerName, outputPath); err != nil {
			return err
		}
	} else {
		// If the output directory does not exist, create it.
		if err = createOutputDirectory(workerName, outputPath); err != nil {
			return err
		}
	}

	// Ensure the removal of the original video file occurs after processing is complete.
	defer removeFile(workerName, videoMsg.FilePath)

	// Attempt to convert the video file up to three times, retrying on failure.
	for i := 1; i <= 3; i++ {
		if videoMsg.Quality != nil {
			// Use the specified video quality for conversion if provided in the message.
			err = pkg.ConvertVideo(videoMsg.FilePath, outputPath, *videoMsg.Quality)
		} else {
			// If no quality is specified, apply the default video quality for conversion.
			err = pkg.ConvertVideo(videoMsg.FilePath, outputPath)
		}

		// Exit the retry loop if conversion is successful.
		if err == nil {
			break
		}

		// On the last attempt (third), log the failure and schedule deletion of the output directory.
		if i == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for video conversion", i)
			RemoveDir(workerName, outputPath)
			return fmt.Errorf("failed to convert video after 3 attempts: %v", err)
		} else {
			// Log a warning if the attempt fails but is not the last one.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for video conversion", i)
		}

		// Clean up the output directory after each failed attempt.
		err = cleanupOutputDirectory(workerName, outputPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// processVideoResolutionsMessage processes a video resolutions message retrieved from the Dead Letter Queue (DLQ).
// It validates the message, checks for the existence of the input file, manages the output directories for each resolution,
// and attempts to convert the video file into multiple resolutions with error handling and retry logic.
func processVideoResolutionsMessage(workerName string, dlqMsg topics.DLQMessage) error {
	var videoResolutionsMsg topics.VideoResolutionsMessage

	// Unmarshal and validate the Kafka message into the VideoResolutionsMessage struct.
	errMsg, err := helper.UnmarshalAndValidate([]byte(dlqMsg.Value), &videoResolutionsMsg)
	if err != nil {
		return fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
	}

	// Ensure the removal of the original video file occurs after processing is complete.
	defer removeFile(workerName, videoResolutionsMsg.FilePath)

	// Define the output path where the converted video resolutions will be stored.
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoResolutionsMsg.NewId)

	// Check if the output directory already exists.
	if _, err = os.Stat(outputPath); !os.IsNotExist(err) {
		// If it exists, clean up any existing files or subdirectories within it.
		if err = cleanupOutputDirectory(workerName, outputPath); err != nil {
			return err
		}
	}

	// Prepare the output directories for each resolution.
	outputPaths := map[string]string{
		"360":  fmt.Sprintf("%s/videos/%s/360", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"480":  fmt.Sprintf("%s/videos/%s/480", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"720":  fmt.Sprintf("%s/videos/%s/720", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
		"1080": fmt.Sprintf("%s/videos/%s/1080", helper.Constants.MediaStorage, videoResolutionsMsg.NewId),
	}

	// Create the output directories for each resolution.
	for _, outputPath := range outputPaths {
		if err = createOutputDirectory(workerName, outputPath); err != nil {
			return err
		}
	}

	// Loop through each resolution and attempt to convert the video with retry logic.
	for res, outputPath := range outputPaths {
		// Retry conversion up to three times.
		for i := 1; i <= 3; i++ {
			// Execute the command to convert the video to the specified resolution.
			err = pkg.ConvertVideoResolutions(videoResolutionsMsg.FilePath, outputPath, res)
			if err == nil {
				break // Exit the loop if conversion is successful.
			}

			// On the last attempt (third), log the failure and schedule deletion of the output directory.
			if i == 3 {
				log.Error().
					Err(err).
					Str("worker", workerName).
					Msgf("Attempt %d failed for video resolution conversion", i)
				RemoveDir(workerName, outputPath)
				return fmt.Errorf("failed to convert video after 3 attempts: %v", err)
			} else {
				// Log a warning if the attempt fails but is not the last one.
				log.Warn().
					Err(err).
					Str("worker", workerName).
					Msgf("Attempt %d failed for video resolution conversion", i)
			}
		}
	}

	return nil
}

// processImageMessage handles the processing of an image message retrieved from the Dead Letter Queue (DLQ).
// It performs message validation, verifies the existence of the input file, and attempts to convert the image
// file up to three times, while logging warnings for failed attempts and errors for the final failure.
func processImageMessage(workerName string, dlqMsg topics.DLQMessage) error {
	var imageMsg topics.ImageMessage

	// Unmarshal the Kafka message from the DLQ and validate it into the ImageMessage struct.
	errMsg, err := helper.UnmarshalAndValidate([]byte(dlqMsg.Value), &imageMsg)
	if err != nil {
		return fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
	}

	// Schedule the removal of the original image file after processing is complete.
	defer removeFile(workerName, imageMsg.FilePath)

	// Construct the output path where the converted image will be saved.
	outputPath := fmt.Sprintf("%s/images/%s.jpeg", helper.Constants.MediaStorage, imageMsg.NewId)

	// Attempt to process the image by executing the conversion command, retrying up to three times if necessary.
	for i := 1; i <= 3; i++ {
		// Call the image processing function, checking for successful conversion.
		if err = pkg.ConvertImage(imageMsg.FilePath, outputPath, "1"); err == nil {
			break // Exit the loop immediately if the conversion is successful.
		}

		// Log an error message if the last attempt (third) fails, and return an error.
		if i == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for image processing: %v", i, err)
			return fmt.Errorf("failed to process image after 3 attempts: %v", err)
		} else {
			// Log a warning if the attempt fails but is not the last one.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for image processing", i)
		}
	}

	// Indicate successful processing by returning nil.
	return nil
}

// processAudioMessage processes an audio message from the Dead Letter Queue (DLQ).
// It validates the message, checks for the existence of the input file,
// and attempts to convert the audio file up to 3 times, logging warnings and errors as needed.
func processAudioMessage(workerName string, dlqMsg topics.DLQMessage) error {
	var audioMsg topics.AudioMessage // Corrected type from ImageMessage to AudioMessage

	// Unmarshal the Kafka message into the AudioMessage struct and validate its contents.
	errMsg, err := helper.UnmarshalAndValidate([]byte(dlqMsg.Value), &audioMsg)
	if err != nil {
		return fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
	}

	// Schedule the removal of the original audio file after processing is complete.
	defer removeFile(workerName, audioMsg.FilePath)

	// Define the output path for the converted audio file.
	outputPath := fmt.Sprintf("%s/audios/%s.mp3", helper.Constants.MediaStorage, audioMsg.NewId)

	// Attempt to execute the audio conversion command up to 3 times.
	for i := 1; i <= 3; i++ {
		// Execute the command for audio conversion using the provided bitrate, if available.
		if audioMsg.Bitrate != nil {
			err = pkg.ConvertAudio(audioMsg.FilePath, outputPath, *audioMsg.Bitrate)
		} else {
			err = pkg.ConvertAudio(audioMsg.FilePath, outputPath) // Call without bitrate
		}

		// If the conversion is successful, exit the loop.
		if err == nil {
			break
		}

		// On the last attempt (third), log the failure and return an error.
		if i == 3 {
			log.Error().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for audio conversion: %v", i, err)
			return fmt.Errorf("failed to convert audio after 3 attempts: %v", err)
		} else {
			// Log a warning if the attempt fails but is not the last one.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for audio conversion", i)
		}
	}

	// Return nil to indicate successful processing of the audio message.
	return nil
}
