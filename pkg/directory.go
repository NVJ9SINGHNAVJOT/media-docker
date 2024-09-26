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

func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		log.Error().Err(err).Str("filePath", filePath).Msg("error deleting file")
		return err
	}
	return nil
}

func DeleteDir(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("error deleting file")
		return err
	}
	return nil
}
