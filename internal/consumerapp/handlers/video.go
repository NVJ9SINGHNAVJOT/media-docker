package handlers

import (
	"path/filepath"

	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

// Video converts an uploaded video to a single-quality HLS stream.
func Video(kafkaMsg []byte) (string, string, error) {
	var videoMsg topics.VideoMessage

	// Unmarshal and Validate the Kafka message into VideoMessage struct
	errMsg, err := validator.UnmarshalAndValidate(kafkaMsg, &videoMsg)
	if err != nil {
		return "", errMsg + " VideoMessage", err
	}

	// Already converted by an earlier delivery of this message.
	if exists(asset.PlaylistPath(videoMsg.NewId)) {
		return videoMsg.NewId, "Video already converted, skipping", nil
	}

	processingDir, err := prepareProcessing(asset.TypeVideo, videoMsg.NewId)
	if err != nil {
		return videoMsg.NewId, "Error preparing processing directory", err
	}

	// Execute the command for video conversion based on the quality
	if videoMsg.Quality != nil {
		err = pkg.ConvertVideo(videoMsg.FilePath, processingDir, *videoMsg.Quality)
	} else {
		err = pkg.ConvertVideo(videoMsg.FilePath, processingDir)
	}
	if err != nil {
		removeProcessing(processingDir)
		return videoMsg.NewId, "Video conversion failed", err
	}

	// Make the playlist and its segments visible in one atomic step.
	if err = asset.PromoteDir(processingDir, asset.HLSDir(videoMsg.NewId)); err != nil {
		removeProcessing(processingDir)
		return videoMsg.NewId, "Error publishing converted video", err
	}

	// The converted stream is live; the raw upload is no longer served.
	pkg.AddToFileDeleteChan(videoMsg.FilePath)

	return videoMsg.NewId, "Video conversion completed successfully", nil
}

// VideoResolutions converts an uploaded video into the full resolution ladder
// and writes the master playlist that lets players switch between the rungs.
func VideoResolutions(kafkaMsg []byte) (string, string, error) {
	var videoMsg topics.VideoResolutionsMessage

	// Unmarshal and Validate the Kafka message into VideoResolutionsMessage struct
	errMsg, err := validator.UnmarshalAndValidate(kafkaMsg, &videoMsg)
	if err != nil {
		return "", errMsg + " VideoResolutionsMessage", err
	}

	// Already converted by an earlier delivery of this message.
	if exists(asset.PlaylistPath(videoMsg.NewId)) {
		return videoMsg.NewId, "Video resolutions already converted, skipping", nil
	}

	processingDir, err := prepareProcessing(asset.TypeVideo, videoMsg.NewId)
	if err != nil {
		return videoMsg.NewId, "Error preparing processing directory", err
	}

	// Produce each rung of the ladder in turn.
	for _, rung := range asset.Ladder {
		rungDir := filepath.Join(processingDir, rung.Name)

		if err = pkg.CreateDir(rungDir); err != nil {
			removeProcessing(processingDir)
			return videoMsg.NewId, "Error creating output directory for resolution " + rung.Name, err
		}

		if err = pkg.ConvertVideoResolutions(videoMsg.FilePath, rungDir, rung.Scale()); err != nil {
			removeProcessing(processingDir)
			return videoMsg.NewId, "Video conversion failed for resolution " + rung.Name, err
		}
	}

	// Without a master playlist the asset's base URL would have no playlist to
	// resolve to, and players would have to be told about each rung separately.
	if err = asset.WriteMasterPlaylist(processingDir); err != nil {
		removeProcessing(processingDir)
		return videoMsg.NewId, "Error writing master playlist", err
	}

	// Publish the whole ladder in one atomic step.
	if err = asset.PromoteDir(processingDir, asset.HLSDir(videoMsg.NewId)); err != nil {
		removeProcessing(processingDir)
		return videoMsg.NewId, "Error publishing converted video resolutions", err
	}

	// The converted stream is live; the raw upload is no longer served.
	pkg.AddToFileDeleteChan(videoMsg.FilePath)

	return videoMsg.NewId, "Video resolution conversion completed successfully", nil
}
