package config

import (
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

// CreateDirSetup sets up the required directories for media storage if they do not exist.
func CreateDirSetup() {
	// UploadStorage directory is created if it does not exist.
	pkg.DirExist(helper.Constants.UploadStorage, true)

	/*
		NOTE: Along with the "/media_docker_files" folder, the "/images" and "/audios" folders
		are also checked and created if they do not exist.
		- Images and audios are stored directly in these folders.
		- Each video, however, will have its own folder inside "/media_docker_files/videos".
	*/

	// Ensure the "images" directory exists inside MediaStorage.
	pkg.DirExist(helper.Constants.MediaStorage+"/images", true)

	// Ensure the "audios" directory exists inside MediaStorage.
	pkg.DirExist(helper.Constants.MediaStorage+"/audios", true)
}
