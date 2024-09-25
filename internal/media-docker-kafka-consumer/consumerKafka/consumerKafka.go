package consumerKafka

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
	ka "github.com/nvj9singhnavjot/media-docker/kafka"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Global KafkaProducer variable
var KafkaProducer *ka.KafkaProducerManager
var KafkaConsumer *ka.KafkaConsumerManager

// ProcessMessage processes the Kafka messages based on the topic
func ProcessMessage(msg kafka.Message, workerName string) {
	var err error
	var topic string
	var id string
	var resMessage string
	var success bool

	// Process messages by topic
	switch msg.Topic {
	case "video":
		topic = "videoResponse"
		id, resMessage, err = processVideoMessage(msg)
	case "videoResolution":
		topic = "videoResolutionResponse"
		id, resMessage, err = processVideoResolutionMessage(msg)
	case "image":
		topic = "imageResponse"
		id, resMessage, err = processImageMessage(msg)
	case "audio":
		topic = "audioResponse"
		id, resMessage, err = processAudioMessage(msg)
	case "deleteFile":
		processDeleteFileMessage(msg)
		return
	default:
		log.Error().
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg("Unkown topic")
		return
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("topic", msg.Topic).
			Int("partition", msg.Partition).
			Str("worker", workerName).
			Str("kafka_message", string(msg.Value)).
			Msg(resMessage)
		if id == "" {
			return
		}
		success = false
	} else {
		success = true
		resMessage = fmt.Sprintf("Successfully processed message for topic: %s", msg.Topic)
	}

	message := serverKafka.KafkaResponseMessage{
		ID:      id,
		Success: success,
		Message: resMessage,
	}
	err = KafkaProducer.Produce(topic, message)
	if err != nil {
		log.Error().
			Err(err).
			Str("topic", topic).
			Str("worker", workerName).
			Any("kafka_message", message).
			Msg("Error while producing message for response")
	}
}

// processVideoMessage processes video conversion and returns the new ID, message, or an error
func processVideoMessage(kafkaMsg kafka.Message) (string, string, error) {
	var videoMsg api.VideoMessage

	// Unmarshal the Kafka message into VideoMessage struct
	if err := json.Unmarshal(kafkaMsg.Value, &videoMsg); err != nil {
		return "", "Failed to unmarshal VideoMessage", err
	}

	// Validate the unmarshaled struct
	if err := helper.ValidateStruct(videoMsg); err != nil {
		return "", "Validation failed for VideoMessage", err
	}

	outputPath := fmt.Sprintf("%s/videos/%s", helper.Constants.MediaStorage, videoMsg.NewId)

	// Create the output directory
	if err := pkg.CreateDir(outputPath); err != nil {
		go pkg.DeleteFile(videoMsg.FilePath)
		return videoMsg.NewId, "error creating output directory", err
	}

	// Prepare the command for video conversion based on the quality
	var cmd *exec.Cmd
	if videoMsg.Quality != nil {
		// Use provided quality
		cmd = pkg.ConvertVideo(videoMsg.FilePath, outputPath, *videoMsg.Quality)
	} else {
		// Use default quality
		cmd = pkg.ConvertVideo(videoMsg.FilePath, outputPath)
	}

	// Execute the command
	if err := cmd.Run(); err != nil {
		return videoMsg.NewId, "Video conversion failed", err
	}

	// Return success: new ID and a success message
	return videoMsg.NewId, "Video conversion completed successfully", nil
}

// processVideoMessage processes video conversion and returns the new ID, message, or an error
func processVideoResolutionMessage(kafkaMsg kafka.Message) (string, string, error) {
	var videoResolutionMsg api.VideoResolutionMessage

	// Unmarshal the Kafka message into VideoMessage struct
	if err := json.Unmarshal(kafkaMsg.Value, &videoResolutionMsg); err != nil {
		return "", "Failed to unmarshal VideoMessage", err
	}

	// Validate the unmarshaled struct
	if err := helper.ValidateStruct(videoResolutionMsg); err != nil {
		return "", "Validation failed for VideoMessage", err
	}

	cmd := pkg.ConvertVideo(videoResolutionMsg.FilePath, videoResolutionMsg.NewId)

	// Execute the command
	if err := cmd.Run(); err != nil {
		return "", "Video conversion failed", err
	}

	// Return success: new ID and a success message
	return videoResolutionMsg.NewId, "Video conversion completed successfully", nil
}
