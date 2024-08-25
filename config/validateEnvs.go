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
}

func ValidateEnvs() (*EnvironmentConfig, error) {

	environment, exist := os.LookupEnv("ENVIRONMENT")
	if !exist {
		return nil, fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exist := os.LookupEnv("ALLOWED_ORIGINS")
	if !exist {
		return nil, fmt.Errorf("allowed origins is not provided")
	}

	serverKey, exist := os.LookupEnv("SERVER_KEY")
	if !exist {
		return nil, fmt.Errorf("server key is not provided")
	}

	port, exist := os.LookupEnv("PORT")
	if !exist {
		return nil, fmt.Errorf("port number is not provided")
	}

	envConfig := &EnvironmentConfig{
		Environment:    environment,
		AllowedOrigins: strings.Split(allowedOrigins, ","),
		ServerKey:      serverKey,
		Port:           port,
	}

	return envConfig, nil
}
