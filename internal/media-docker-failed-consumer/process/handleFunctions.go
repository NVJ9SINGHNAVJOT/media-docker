package process

import (
	"fmt"
	"os"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
)

// processVideoMessage processes a video message retrieved from the "failed-letter-queue".
// It validates the message, checks for the existence of the input file, manages the output directory,
// and attempts to convert the video file up to three times, implementing error handling and retry logic as necessary.
func processVideoMessage(workerName string, dlqMsg topics.DLQMessage) (string, error) {
	var videoMsg topics.VideoMessage

	// Unmarshal and validate the Kafka message into the VideoMessage struct.
	errMsg, err := validator.UnmarshalAndValidate([]byte(dlqMsg.Value), &videoMsg)
	if err != nil {
		return "", fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
	}

	// Define the output path where the converted video will be stored.
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoMsg.NewId)

	// Check if the output directory already exists.
	if _, err = os.Stat(outputPath); !os.IsNotExist(err) {
		// If it exists, clean up any existing files or subdirectories within it.
		if err = cleanupOutputDirectory(workerName, outputPath); err != nil {
			return videoMsg.NewId, err
		}
	} else {
		// If the output directory does not exist, create it.
		if err = createOutputDirectory(workerName, outputPath); err != nil {
			return videoMsg.NewId, err
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
			return videoMsg.NewId, fmt.Errorf("failed to convert video after 3 attempts: %v", err)
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
			return videoMsg.NewId, err
		}
	}

	return videoMsg.NewId, nil
}

// processVideoResolutionsMessage processes a video resolutions message retrieved from the "failed-letter-queue".
// It validates the message, checks for the existence of the input file, manages the output directories for each resolution,
// and attempts to convert the video file into multiple resolutions with error handling and retry logic.
func processVideoResolutionsMessage(workerName string, dlqMsg topics.DLQMessage) (string, error) {
	var videoResolutionsMsg topics.VideoResolutionsMessage

	// Unmarshal and validate the Kafka message into the VideoResolutionsMessage struct.
	errMsg, err := validator.UnmarshalAndValidate([]byte(dlqMsg.Value), &videoResolutionsMsg)
	if err != nil {
		return "", fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
	}

	// Ensure the removal of the original video file occurs after processing is complete.
	defer removeFile(workerName, videoResolutionsMsg.FilePath)

	// Define the output path where the converted video resolutions will be stored.
	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoResolutionsMsg.NewId)

	// Check if the output directory already exists.
	if _, err = os.Stat(outputPath); !os.IsNotExist(err) {
		// If it exists, clean up any existing files or subdirectories within it.
		if err = cleanupOutputDirectory(workerName, outputPath); err != nil {
			return videoResolutionsMsg.NewId, err
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
			return videoResolutionsMsg.NewId, err
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
				return videoResolutionsMsg.NewId, fmt.Errorf("failed to convert video after 3 attempts: %v", err)
			} else {
				// Log a warning if the attempt fails but is not the last one.
				log.Warn().
					Err(err).
					Str("worker", workerName).
					Msgf("Attempt %d failed for video resolution conversion", i)
			}
		}
	}

	return videoResolutionsMsg.NewId, nil
}

// processImageMessage handles the processing of an image message retrieved from the "failed-letter-queue".
// It performs message validation, verifies the existence of the input file, and attempts to convert the image
// file up to three times, while logging warnings for failed attempts and errors for the final failure.
func processImageMessage(workerName string, dlqMsg topics.DLQMessage) (string, error) {
	var imageMsg topics.ImageMessage

	// Unmarshal the Kafka message from the DLQ and validate it into the ImageMessage struct.
	errMsg, err := validator.UnmarshalAndValidate([]byte(dlqMsg.Value), &imageMsg)
	if err != nil {
		return "", fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
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
			return imageMsg.NewId, fmt.Errorf("failed to process image after 3 attempts: %v", err)
		} else {
			// Log a warning if the attempt fails but is not the last one.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for image processing", i)
		}
	}

	// Indicate successful processing by returning nil.
	return imageMsg.NewId, nil
}

// processAudioMessage processes an audio message from the "failed-letter-queue".
// It validates the message, checks for the existence of the input file,
// and attempts to convert the audio file up to 3 times, logging warnings and errors as needed.
func processAudioMessage(workerName string, dlqMsg topics.DLQMessage) (string, error) {
	var audioMsg topics.AudioMessage // Corrected type from ImageMessage to AudioMessage

	// Unmarshal the Kafka message into the AudioMessage struct and validate its contents.
	errMsg, err := validator.UnmarshalAndValidate([]byte(dlqMsg.Value), &audioMsg)
	if err != nil {
		return "", fmt.Errorf("error during message unmarshalling and validation: %s, %v", errMsg, err)
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
			return audioMsg.NewId, fmt.Errorf("failed to convert audio after 3 attempts: %v", err)
		} else {
			// Log a warning if the attempt fails but is not the last one.
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Msgf("Attempt %d failed for audio conversion", i)
		}
	}

	// Return nil to indicate successful processing of the audio message.
	return audioMsg.NewId, nil
}
