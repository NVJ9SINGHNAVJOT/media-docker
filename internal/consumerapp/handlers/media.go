package handlers

import (
	"path/filepath"

	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

// defaultImageCompression is the ffmpeg -q:v value used when a request does not
// specify one. 1 is the highest quality the scale allows.
const defaultImageCompression = 1

// Image compresses an uploaded image to JPEG.
func Image(kafkaMsg []byte) (string, string, error) {
	var imageMsg topics.ImageMessage

	// Unmarshal and Validate the Kafka message into ImageMessage struct
	errMsg, err := validator.UnmarshalAndValidate(kafkaMsg, &imageMsg)
	if err != nil {
		return "", errMsg + " ImageMessage", err
	}

	finalPath, err := asset.ConvertedPath(asset.TypeImage, imageMsg.NewId)
	if err != nil {
		return imageMsg.NewId, "Error resolving output path", err
	}

	// Already converted by an earlier delivery of this message.
	if exists(finalPath) {
		return imageMsg.NewId, "Image already converted, skipping", nil
	}

	processingDir, err := prepareProcessing(asset.TypeImage, imageMsg.NewId)
	if err != nil {
		return imageMsg.NewId, "Error preparing processing directory", err
	}

	compression := defaultImageCompression
	if imageMsg.Compression != nil {
		compression = *imageMsg.Compression
	}

	tmpPath := filepath.Join(processingDir, asset.ConvertedImageName)

	if err = pkg.ConvertImage(imageMsg.FilePath, tmpPath, itoa(compression)); err != nil {
		removeProcessing(processingDir)
		return imageMsg.NewId, "Image conversion failed", err
	}

	if err = asset.PromoteFile(tmpPath, finalPath); err != nil {
		removeProcessing(processingDir)
		return imageMsg.NewId, "Error publishing converted image", err
	}

	removeProcessing(processingDir)

	return imageMsg.NewId, "Image conversion completed successfully", nil
}

// Audio converts an uploaded audio file to MP3 at the requested bitrate.
func Audio(kafkaMsg []byte) (string, string, error) {
	var audioMsg topics.AudioMessage

	// Unmarshal and Validate the Kafka message into AudioMessage struct
	errMsg, err := validator.UnmarshalAndValidate(kafkaMsg, &audioMsg)
	if err != nil {
		return "", errMsg + " AudioMessage", err
	}

	finalPath, err := asset.ConvertedPath(asset.TypeAudio, audioMsg.NewId)
	if err != nil {
		return audioMsg.NewId, "Error resolving output path", err
	}

	// Already converted by an earlier delivery of this message.
	if exists(finalPath) {
		return audioMsg.NewId, "Audio already converted, skipping", nil
	}

	processingDir, err := prepareProcessing(asset.TypeAudio, audioMsg.NewId)
	if err != nil {
		return audioMsg.NewId, "Error preparing processing directory", err
	}

	tmpPath := filepath.Join(processingDir, asset.ConvertedAudioName)

	if audioMsg.Bitrate != nil {
		err = pkg.ConvertAudio(audioMsg.FilePath, tmpPath, *audioMsg.Bitrate)
	} else {
		err = pkg.ConvertAudio(audioMsg.FilePath, tmpPath)
	}
	if err != nil {
		removeProcessing(processingDir)
		return audioMsg.NewId, "Audio conversion failed", err
	}

	if err = asset.PromoteFile(tmpPath, finalPath); err != nil {
		removeProcessing(processingDir)
		return audioMsg.NewId, "Error publishing converted audio", err
	}

	removeProcessing(processingDir)

	return audioMsg.NewId, "Audio conversion completed successfully", nil
}
