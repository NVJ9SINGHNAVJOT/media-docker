package pkg

import (
	"os"

	"github.com/rs/zerolog/log"
)

func DeleteFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Error().Msg("error deleting file, " + err.Error())
		return
	}
}
