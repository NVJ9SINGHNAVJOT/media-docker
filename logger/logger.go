// Package logger provides functions for custom logging used in the Media Docker project.
// This package is used to log structured error messages, specifically for Kafka message processing,
// with detailed metadata and optional error details such as missing ID information.
package logger

import (
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// LogErrorWithMissingID logs an error message with detailed Kafka message metadata and
// highlights the case where the ID is missing from the message processing.
//
// Parameters:
// - err: The error that occurred during message processing.
// - workerName: The name of the worker processing the message.
// - msg: The Kafka message being processed, which includes details like topic, partition, offset, etc.
// - resMessage: A custom message that describes the error context or result.
func LogErrorWithMissingID(err error, workerName string, msg kafka.Message, resMessage string) {
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("message_details", map[string]interface{}{
			"topic":         msg.Topic,
			"partition":     msg.Partition,
			"offset":        msg.Offset,
			"highWaterMark": msg.HighWaterMark,
			"value":         string(msg.Value),
			"time":          msg.Time,
		}).
		Str("id", "ID not returned from message processing"). // Log missing ID error
		Msg(resMessage)
}

// LogErrorWithKafkaMessage logs a standard error message with detailed Kafka message metadata.
//
// Parameters:
// - err: The error that occurred during message processing.
// - workerName: The name of the worker processing the message.
// - msg: The Kafka message being processed, which includes details like topic, partition, offset, etc.
// - resMessage: A custom message that describes the error context or result.
func LogErrorWithKafkaMessage(err error, workerName string, msg kafka.Message, resMessage string) {
	log.Error().
		Err(err).
		Str("worker", workerName).
		Interface("message_details", map[string]interface{}{
			"topic":         msg.Topic,
			"partition":     msg.Partition,
			"offset":        msg.Offset,
			"highWaterMark": msg.HighWaterMark,
			"value":         string(msg.Value),
			"time":          msg.Time,
		}).
		Msg(resMessage)
}

// LogUnknownTopic logs a standard error message with detailed Kafka message metadata,
// typically used when the topic of the Kafka message is unknown or unrecognized.
//
// Parameters:
// - workerName: The name of the worker processing the message.
// - msg: The Kafka message being processed, which includes details like topic, partition, offset, etc.
func LogUnknownTopic(workerName string, msg kafka.Message) {
	log.Error().
		Str("worker", workerName).
		Interface("message_details", map[string]interface{}{
			"topic":         msg.Topic,
			"partition":     msg.Partition,
			"offset":        msg.Offset,
			"highWaterMark": msg.HighWaterMark,
			"value":         string(msg.Value),
			"time":          msg.Time,
		}).
		Msg("Unknown topic")
}
