package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

/*
	NOTE: short notation
	MDC = media-docker-client
	MDS = media-docker-server
*/

type MDCEnvironmentConfig struct {
	ENVIRONMENT            string
	ALLOWED_ORIGINS_CLIENT []string
	CLIENT_PORT            string
}

type MDSEnvironmentConfig struct {
	ENVIRONMENT                       string
	ALLOWED_ORIGINS_SERVER            []string
	SERVER_KEY                        string
	VIDEO_WORKER_POOL_SIZE            int
	VIDEO_RESOLUTION_WORKER_POOL_SIZE int
	IMAGE_WORKER_POOL_SIZE            int
	AUDIO_WORKER_POOL_SIZE            int
	BASE_URL                          string
	SERVER_PORT                       string
}

var MDCenvs = MDCEnvironmentConfig{}
var MDSenvs = MDSEnvironmentConfig{}

func ValidateMDCenvs() error {

	environment, exist := os.LookupEnv("ENVIRONMENT")
	if !exist {
		return fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exist := os.LookupEnv("ALLOWED_ORIGINS_CLIENT")
	if !exist {
		return fmt.Errorf("allowed origins are not provided")
	}

	port, exist := os.LookupEnv("CLIENT_PORT")
	if !exist {
		return fmt.Errorf("client port is not provided")
	}

	MDCenvs.ENVIRONMENT = environment
	MDCenvs.ALLOWED_ORIGINS_CLIENT = strings.Split(allowedOrigins, ",")
	MDCenvs.CLIENT_PORT = port

	return nil
}

func ValidateMDSenvs() error {

	environment, exist := os.LookupEnv("ENVIRONMENT")
	if !exist {
		return fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exist := os.LookupEnv("ALLOWED_ORIGINS_SERVER")
	if !exist {
		return fmt.Errorf("allowed origins are not provided")
	}

	serverKey, exist := os.LookupEnv("SERVER_KEY")
	if !exist {
		return fmt.Errorf("server key is not provided")
	}

	videoWorkerPoolSizeStr, exist := os.LookupEnv("VIDEO_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("video worker pool size is not provided")
	}
	videoWorkerPoolSize, err := strconv.Atoi(videoWorkerPoolSizeStr)
	if err != nil {
		return fmt.Errorf("invalid video worker pool size: %v", err)
	}

	videoResolutionWorkerPoolSizeStr, exist := os.LookupEnv("VIDEO_RESOLUTION_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("video resolution worker pool size is not provided")
	}
	videoResolutionWorkerPoolSize, err := strconv.Atoi(videoResolutionWorkerPoolSizeStr)
	if err != nil {
		return fmt.Errorf("invalid video resolution worker pool size: %v", err)
	}

	imageWorkerPoolSizeStr, exist := os.LookupEnv("IMAGE_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("image worker pool size is not provided")
	}
	imageWorkerPoolSize, err := strconv.Atoi(imageWorkerPoolSizeStr)
	if err != nil {
		return fmt.Errorf("invalid image worker pool size: %v", err)
	}

	audioWorkerPoolSizeStr, exist := os.LookupEnv("AUDIO_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("audio worker pool size is not provided")
	}
	audioWorkerPoolSize, err := strconv.Atoi(audioWorkerPoolSizeStr)
	if err != nil {
		return fmt.Errorf("invalid audio worker pool size: %v", err)
	}

	// Validate worker pool sizes
	if videoWorkerPoolSize < 1 {
		return fmt.Errorf("minimum size required for video channel workers is 1")
	}
	if videoResolutionWorkerPoolSize < 1 {
		return fmt.Errorf("minimum size required for video resolution channel workers is 1")
	}
	if imageWorkerPoolSize < 1 {
		return fmt.Errorf("minimum size required for image channel workers is 1")
	}
	if audioWorkerPoolSize < 1 {
		return fmt.Errorf("minimum size required for audio channel workers is 1")
	}

	port, exist := os.LookupEnv("SERVER_PORT")
	if !exist {
		return fmt.Errorf("server port is not provided")
	}

	baseUrl, exist := os.LookupEnv("BASE_URL")
	if !exist {
		return fmt.Errorf("base URL is not provided")
	}

	MDSenvs.ENVIRONMENT = environment
	MDSenvs.ALLOWED_ORIGINS_SERVER = strings.Split(allowedOrigins, ",")
	MDSenvs.SERVER_KEY = serverKey
	MDSenvs.SERVER_PORT = port
	MDSenvs.BASE_URL = baseUrl
	MDSenvs.VIDEO_WORKER_POOL_SIZE = videoWorkerPoolSize
	MDSenvs.VIDEO_RESOLUTION_WORKER_POOL_SIZE = videoResolutionWorkerPoolSize
	MDSenvs.IMAGE_WORKER_POOL_SIZE = imageWorkerPoolSize
	MDSenvs.AUDIO_WORKER_POOL_SIZE = audioWorkerPoolSize

	return nil
}
