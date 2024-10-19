package helper

type constConfig struct {
	UploadStorage string
	MediaStorage  string
	MaxFileSize   map[string]int64
}

// NOTE: do not change these values, project will break
var Constants = &constConfig{
	UploadStorage: "uploadStorage",
	MediaStorage:  "media_docker_files",
	MaxFileSize: map[string]int64{
		"videoFile": 1024 * 1024 * 1000, // 1 GB
		"imageFile": 1024 * 1024 * 50,   // 50 MB
		"audioFile": 1024 * 1024 * 50,   // 50 MB
	},
}
