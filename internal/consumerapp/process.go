package consumerapp

import (
	"time"

	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// messageProcessor builds the per-message callback handed to the Kafka consumer
// manager.
//
// A service whose failures must not be retried through the dead-letter queue
// supplies a RawHandler and is wired straight through; everything else gets the
// standard wrapper that guards the topic and routes failures to the DLQ.
func messageProcessor(cfg Config) func(msg kafka.Message, workerName string) {
	if cfg.RawHandler != nil {
		return cfg.RawHandler
	}

	return func(msg kafka.Message, workerName string) {
		// Each service owns exactly one topic, so anything else on this reader
		// means a misconfigured subscription rather than an unhandled type.
		if msg.Topic != cfg.Topic {
			logger.LogUnknownTopic(workerName, msg)
			return
		}

		newId, resMessage, err := cfg.Handler(msg.Value)
		if err != nil {
			handleErrorResponse(msg, workerName, newId, resMessage, err)
		}
	}
}

// handleErrorResponse routes a failed message to the dead-letter queue.
//
// The message is logged with its Kafka metadata and re-published as a
// DLQMessage on "failed-letter-queue", where the failed consumer will retry the
// conversion. The asset id is carried along when it can be determined, either
// from the handler or from the raw payload, so a retry can be traced back.
//
// NOTE: Since v4 a failure here is not user-visible damage. The raw upload is
// still in place and the asset's URL keeps serving it, so the worst outcome is
// that the asset is never upgraded to its converted form.
func handleErrorResponse(msg kafka.Message, workerName, newId, resMessage string, err error) {
	logger.LogErrorWithKafkaMessage(err, workerName, msg, resMessage)

	// NOTE: Failed messages are sent to the "failed-letter-queue" topic,
	// enabling further processing and reducing retry load on the main consumption service.
	//
	// Create a DLQMessage struct with error details and original message information.
	dlqMessage := topics.DLQMessage{
		OriginalTopic:  msg.Topic,
		Partition:      msg.Partition,
		Offset:         msg.Offset,
		HighWaterMark:  msg.HighWaterMark,
		Value:          string(msg.Value),
		ErrorDetails:   err.Error(),
		ProcessingTime: msg.Time,
		ErrorTime:      time.Now(),
		Worker:         workerName,
		CustomMessage:  resMessage,
	}

	// If newId is empty, attempt to extract it from the message value.
	if newId == "" {
		extracted, extractErr := validator.ExtractNewId(msg.Value)
		if extractErr == nil {
			newId = extracted
		}
	}

	if newId != "" {
		dlqMessage.NewId = &newId
	}

	// Attempt to produce the DLQ message to the "failed-letter-queue" topic.
	//
	// A failure here means the asset is simply never retried: it keeps serving
	// the raw upload it was created with, so this log is the whole recovery path.
	if err = kafkahandler.KafkaProducer.Produce("failed-letter-queue", dlqMessage); err != nil {
		log.Error().
			Err(err).
			Str("worker", workerName).
			Interface("dlq_message", dlqMessage).
			Msg("Error producing message to failed-letter-queue.")
	}
}
