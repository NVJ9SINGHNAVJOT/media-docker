package helper

type ConstConfig struct {
	UploadStorage string
	MediaStorage  string
	MaxFileSize   int64
}

// NOTE: do not change these values, project will break
var Constants = &ConstConfig{
	UploadStorage: "uploadStorage",
	MediaStorage:  "media_docker_files",
	MaxFileSize:   1024 * 1024 * 1000,
}
