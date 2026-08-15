package handlers

import (
	"fmt"
	"os"

	"github.com/nvj9singhnavjot/media-docker/api"
	"github.com/nvj9singhnavjot/media-docker/logger"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// DeleteFile removes an asset from media storage.
//
// Because v4 stores every asset -- of any type, converted or not -- as a single
// directory, deletion needs no per-type branching: one RemoveAll takes the raw
// upload, the converted output and the metadata together. This is why a single
// small service can own deletion for every media type.
//
// This is a RawHandler: the failed consumer has no handler for this topic, so a
// failure is logged rather than routed to the dead-letter queue.
func DeleteFile(msg kafka.Message, workerName string) {
	var deleteFileMsg api.DeleteFileRequest

	// Unmarshal and Validate the Kafka message into DeleteFileRequest struct
	errMsg, err := validator.UnmarshalAndValidate(msg.Value, &deleteFileMsg)
	if err != nil {
		logger.LogErrorWithKafkaMessage(err, workerName, msg, errMsg+" DeleteFileMessage")
		return
	}

	if !asset.IsStorageType(deleteFileMsg.Type) {
		logger.LogErrorWithKafkaMessage(
			fmt.Errorf("unknown media type %q", deleteFileMsg.Type),
			workerName, msg, "Cannot delete asset of unknown type")
		return
	}

	path := asset.Dir(deleteFileMsg.Type, deleteFileMsg.Id)

	if err = os.RemoveAll(path); err != nil {
		logger.LogErrorWithKafkaMessage(
			err,
			workerName,
			msg,
			fmt.Sprintf("Error while deleting %s asset, id: %s, path: %s", deleteFileMsg.Type, deleteFileMsg.Id, path))
		return
	}

	log.Info().
		Str("worker", workerName).
		Str("type", deleteFileMsg.Type).
		Str("id", deleteFileMsg.Id).
		Msg("Asset deleted.")
}
