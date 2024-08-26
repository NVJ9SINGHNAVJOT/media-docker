package config

import (
	"fmt"
	"os"
	"strings"
)

type EnvironmentConfig struct {
	Environment    string
	AllowedOrigins []string
	ServerKey      string
	Port           string
	BASE_URL       string
}

var Envs = EnvironmentConfig{}

func ValidateEnvs() error {

	environment, exist := os.LookupEnv("ENVIRONMENT")
	if !exist {
		return fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exist := os.LookupEnv("ALLOWED_ORIGINS")
	if !exist {
		return fmt.Errorf("allowed origins is not provided")
	}

	serverKey, exist := os.LookupEnv("SERVER_KEY")
	if !exist {
		return fmt.Errorf("server key is not provided")
	}

	port, exist := os.LookupEnv("PORT")
	if !exist {
		return fmt.Errorf("port number is not provided")
	}

	baseUrl, exist := os.LookupEnv("BASE_URL")
	if !exist {
		return fmt.Errorf("port number is not provided")
	}

	Envs.Environment = environment
	Envs.AllowedOrigins = strings.Split(allowedOrigins, ",")
	Envs.ServerKey = serverKey
	Envs.Port = port
	Envs.BASE_URL = baseUrl

	return nil
}
