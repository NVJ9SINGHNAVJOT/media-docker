package config

import (
	"fmt"
	"os"
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
	VIDEO_WORKER_POOL_SIZE            string
	VIDEO_RESOLUTION_WORKER_POOL_SIZE string
	IMAGE_WORKER_POOL_SIZE            string
	AUDIO_WORKER_POOL_SIZE            string
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
		return fmt.Errorf("allowed origins is not provided")
	}

	port, exist := os.LookupEnv("CLIENT_PORT")
	if !exist {
		return fmt.Errorf("port number is not provided")
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
		return fmt.Errorf("allowed origins is not provided")
	}

	serverKey, exist := os.LookupEnv("SERVER_KEY")
	if !exist {
		return fmt.Errorf("server key is not provided")
	}

	videoWorkerPoolSize, exist := os.LookupEnv("VIDEO_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("video worker pool size is no provided")
	}

	videoResolutionWorkerPoolSize, exist := os.LookupEnv("VIDEO_RESOLUTION_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("videoResolution worker pool size is no provided")
	}

	imageWorkerPoolSize, exist := os.LookupEnv("IMAGE_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("image worker pool size is no provided")
	}

	audioWorkerPoolSize, exist := os.LookupEnv("AUDIO_WORKER_POOL_SIZE")
	if !exist {
		return fmt.Errorf("audio worker pool size is no provided")
	}

	port, exist := os.LookupEnv("SERVER_PORT")
	if !exist {
		return fmt.Errorf("port number is not provided")
	}

	baseUrl, exist := os.LookupEnv("BASE_URL")
	if !exist {
		return fmt.Errorf("port number is not provided")
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
