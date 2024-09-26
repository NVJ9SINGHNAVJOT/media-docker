package pkg

import (
	"os"
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
	return os.MkdirAll(outputPath, os.ModePerm)
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
