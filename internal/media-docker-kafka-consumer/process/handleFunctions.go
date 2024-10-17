package process

import (
	"fmt"
	"os"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/segmentio/kafka-go"
)

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
			pkg.AddToDirDeleteChan(fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoResolutionsMsg.NewId))
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
