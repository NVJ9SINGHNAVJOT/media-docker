package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

func CreateDir(outputPath string) error {
	err := os.MkdirAll(outputPath, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func CreateDirs(outputPaths []string) error {
	for _, v := range outputPaths {
		err := CreateDir(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Error().Msg("error deleting file, error: " + err.Error())
		return
	}
}
