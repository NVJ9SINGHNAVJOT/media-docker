package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetUpLogger(environment string) {
	if environment == "development" {
		cw := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
		
		cw.FormatFieldName = func(i interface{}) string {
			return fmt.Sprintf("\n  %s=", i)
		}
		
		cw.FormatFieldValue = func(i interface{}) string {
			s := fmt.Sprintf("%s", i)
			if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
				var obj interface{}
				if err := json.Unmarshal([]byte(s), &obj); err == nil {
					b, _ := json.MarshalIndent(obj, "    ", "  ")
					return fmt.Sprintf("\n    %s", string(b))
				}
			}
			return s
		}
		log.Logger = log.Output(cw).Level(zerolog.DebugLevel)
		return
	}
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}
