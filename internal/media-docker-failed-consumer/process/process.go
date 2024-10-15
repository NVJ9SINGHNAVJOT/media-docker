// process package contains Kafka message processing functions for media-docker-failed-consumer.
package process

import (
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	log.Debug().Any("kafka_message", string(msg.Value)).Msg("TODO: processing of kafka message")
}
