package config

import (
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

// CreateDirSetup ensures that the required directories for media storage are created if they do not already exist.
// It sets up directories for storing chunked files and media files for videos, images, and audios.
func CreateDirSetup() {
	// Create directories within UploadStorage for chunked "videos", "images", and "audios" files, if they don't exist.
	pkg.DirExist(helper.Constants.UploadStorage+"/videos", true)
	pkg.DirExist(helper.Constants.UploadStorage+"/images", true)
	pkg.DirExist(helper.Constants.UploadStorage+"/audios", true)

	// Ensure the "videos" directory exists within MediaStorage.
	pkg.DirExist(helper.Constants.MediaStorage+"/videos", true)

	// Ensure the "images" directory exists within MediaStorage.
	pkg.DirExist(helper.Constants.MediaStorage+"/images", true)

	// Ensure the "audios" directory exists within MediaStorage.
	pkg.DirExist(helper.Constants.MediaStorage+"/audios", true)
}
