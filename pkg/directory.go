package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

func DirExist(dirPath string) (bool, error) {
	_, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

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
		log.Error().Str("error", err.Error()).Msg("error deleting file")
	}
}
