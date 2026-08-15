// Package process implements the retry logic of media-docker-failed-consumer.
//
// It consumes the dead-letter queue and re-runs the conversion that failed. It
// deliberately calls the very same handlers the primary consumers use rather
// than keeping a parallel implementation: those handlers are idempotent, write
// through a scratch directory, and leave the raw upload untouched on failure,
// which is exactly what a retry needs. Before v4 this package duplicated every
// conversion, and the two copies were free to drift apart.
package process

import (
	"time"

	"github.com/nvj9singhnavjot/media-docker/internal/consumerapp/handlers"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/topics"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Retry settings for a message pulled off the dead-letter queue.
const (
	retryAttempts = 3
	retryBackoff  = 2 * time.Second
)

// conversion re-runs the work a failed message describes. It is the very same
// function the primary consumer used, not a parallel implementation.
type conversion func(payload []byte) (newId string, resMessage string, err error)

// topicHandlers maps the topic a failed message came from to its conversion.
var topicHandlers = map[string]conversion{
	"video":             handlers.Video,
	"video-resolutions": handlers.VideoResolutions,
	"image":             handlers.Image,
	"audio":             handlers.Audio,
}

// ProcessMessage processes a message from the dead-letter queue.
//
// If the message unmarshals and validates, the conversion it describes is
// retried. If it does not, there is nothing to act on and it is logged.
func ProcessMessage(msg kafka.Message, workerName string) {
	var dlqMsg topics.DLQMessage

	// Unmarshal and validate the message.
	errmsg, err := validator.UnmarshalAndValidate(msg.Value, &dlqMsg)

	if err == nil {
		handleDLQMessage(dlqMsg, workerName)
		return
	}

	// The message is not a DLQMessage, so there is nothing to retry and nothing
	// to recover from it. Whatever asset it referred to keeps serving its raw
	// upload; only the retry is lost.
	logger.LogErrorWithKafkaMessage(err, workerName, msg, errmsg+" DLQMessage")
}

// handleDLQMessage retries the conversion a failed message describes.
//
// A permanently failed conversion is no longer a broken asset. The raw upload is
// still in place and the asset's URL keeps serving it, so exhausting the retries
// means the asset stays in its original quality rather than that it is
// unavailable.
func handleDLQMessage(dlqMsg topics.DLQMessage, workerName string) {
	convert, exists := topicHandlers[dlqMsg.OriginalTopic]
	if !exists {
		// The topic passed struct validation but has no handler, which can only
		// mean the validation tag and this map have drifted apart.
		log.Error().
			Str("worker", workerName).
			Interface("dlq_message", dlqMsg).
			Msg("No handler for original topic of DLQMessage.")
		return
	}

	log.Info().
		Str("worker", workerName).
		Interface("dlq_message", dlqMsg).
		Msg("DLQMessage received.")

	newId, resMessage, err := retryConversion(convert, dlqMsg, workerName)

	if err == nil {
		log.Info().
			Str("worker", workerName).
			Str("newId", newId).
			Interface("dlq_message", dlqMsg).
			Msg("DLQMessage processing completed successfully.")
		return
	}

	// INFO: This was the last attempt at converting this asset. The raw upload
	// remains in place and continues to be served, so the asset stays usable at
	// its original quality rather than becoming unavailable.
	log.Error().
		Err(err).
		Str("worker", workerName).
		Str("detail", resMessage).
		Interface("dlq_message", dlqMsg).
		Msg("Failed to process DLQMessage, asset keeps serving its raw upload.")
}

// retryConversion re-runs a conversion, giving it a few attempts before giving up.
func retryConversion(convert conversion, dlqMsg topics.DLQMessage, workerName string) (string, string, error) {
	var newId, resMessage string
	var err error

	payload := []byte(dlqMsg.Value)

	for attempt := 1; attempt <= retryAttempts; attempt++ {
		newId, resMessage, err = convert(payload)
		if err == nil {
			return newId, resMessage, nil
		}

		if attempt < retryAttempts {
			log.Warn().
				Err(err).
				Str("worker", workerName).
				Str("detail", resMessage).
				Msgf("Attempt %d/%d failed for %s retry, retrying", attempt, retryAttempts, dlqMsg.OriginalTopic)

			time.Sleep(retryBackoff)
		}
	}

	return newId, resMessage, err
}
