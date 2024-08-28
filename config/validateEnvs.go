package config

import (
	"fmt"
	"os"
	"strings"
)

type MDSEnvironmentConfig struct {
	Environment    string
	AllowedOrigins []string
	ServerKey      string
	Port           string
	BASE_URL       string
}

var MDSenvs = MDSEnvironmentConfig{}

func ValidateMDSenvs() error {

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

	MDSenvs.Environment = environment
	MDSenvs.AllowedOrigins = strings.Split(allowedOrigins, ",")
	MDSenvs.ServerKey = serverKey
	MDSenvs.Port = port
	MDSenvs.BASE_URL = baseUrl

	return nil
}

type MDCEnvironmentConfig struct {
	Environment    string
	AllowedOrigins []string
	Port           string
}

var MDCenvs = MDCEnvironmentConfig{}

func ValidateMDCenvs() error {

	environment, exist := os.LookupEnv("ENVIRONMENT")
	if !exist {
		return fmt.Errorf("environment is not provided")
	}

	allowedOrigins, exist := os.LookupEnv("ALLOWED_ORIGINS")
	if !exist {
		return fmt.Errorf("allowed origins is not provided")
	}

	port, exist := os.LookupEnv("PORT")
	if !exist {
		return fmt.Errorf("port number is not provided")
	}

	MDCenvs.Environment = environment
	MDCenvs.AllowedOrigins = strings.Split(allowedOrigins, ",")
	MDCenvs.Port = port

	return nil
}
