// process package contains Kafka message processing functions for media-docker-failed-consumer.
package process

import (
	"fmt"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// fileTypeMap maps topics to their corresponding file types.
var fileTypeMap = map[string]string{
	"video":             "video",
	"video-resolutions": "videoResolutions",
	"image":             "image",
	"audio":             "audio",
}

// getFileTypeByTopic returns the file type based on the topic.
// If the topic is not found, it returns an error.
func getFileTypeByTopic(topic string) (fileType string, err error) {
	// Lookup the topic in the map
	fileType, found := fileTypeMap[topic]
	if !found {
		err = fmt.Errorf("topic %s not found", topic) // Return an error if topic is not recognized
	}
	return fileType, err
}

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var dlqMsg topics.DLQMessage

	errmsg, err := helper.UnmarshalAndValidate(msg.Value, &dlqMsg)

	if err == nil {
		handleDLQMessage(dlqMsg)
		return
	}

	// Now try to get newId from message
	newId, originalTopic, extractErr := helper.ExtractNewIdAndOriginalTopic(msg.Value)
	if extractErr == nil {
		fileType, typeErr := getFileTypeByTopic(originalTopic)
		if typeErr == nil {
			logger.LogErrorWithKafkaMessage(err, workerName, msg, errmsg+" DLQMessage")
			kafkahandler.SendConsumerResponse(workerName, newId, fileType, "failed")
			return
		}
	}

	// NOTE: No response will be sent to "media-docker-files-response",
	// leaving client backend services unnotified.
	log.Error().
		Err(err).
		Interface("dlq", dlqMsg).
		Str("failed", "Failed to get newId and originalTopic").
		Msg(errmsg + " DLQMessage")
}

func handleDLQMessage(dlqMsg topics.DLQMessage) {
	log.Debug().Any("dlq", dlqMsg).Msg("in progress")
}
