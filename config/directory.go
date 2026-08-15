package config

import (
	"path/filepath"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
)

// storageTypes lists every category that gets a directory under both storage roots.
var storageTypes = []string{
	asset.TypeVideo, asset.TypeImage, asset.TypeAudio, asset.TypeDocument, asset.TypeOther,
}

// CreateDirSetup ensures the storage directory tree exists, creating anything missing.
//
// This is the media-docker-server's responsibility: it is the only service that
// creates assets, so it is the only one that needs to bootstrap the tree.
// Consumers write into asset directories that already exist and merely verify
// the storage roots are present.
func CreateDirSetup() {
	for _, mediaType := range storageTypes {
		dir := asset.TypeDir(mediaType)

		// Staging directories for chunked uploads.
		pkg.DirExist(filepath.Join(helper.Constants.UploadStorage, dir), true)

		// Publicly served asset directories.
		pkg.DirExist(filepath.Join(helper.Constants.MediaStorage, dir), true)
	}
}
