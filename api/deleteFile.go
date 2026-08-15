package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/validator"
)

// DeleteFileRequest is the payload of both the delete API and the "delete-file"
// Kafka topic, which is why it is exported: the delete consumer unmarshals the
// same struct.
type DeleteFileRequest struct {
	Id   string `json:"id" validate:"required,uuid4"`
	Type string `json:"type" validate:"required,oneof=image video audio document other"`
}

// DeleteFile queues an asset for removal.
//
// Because every asset is a single directory regardless of type, deletion is
// type-agnostic and the consumer simply removes that directory.
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	var req DeleteFileRequest

	// Parse the JSON request and populate the DeleteFileRequest struct.
	if err := validator.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	exist, err := pkg.DirOrFileExist(asset.Dir(req.Type, req.Id))
	if err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid file for deleting", err)
		return
	}

	if !exist {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "file doesn't exist for deleting", nil)
		return
	}

	if err := kafkahandler.KafkaProducer.Produce("delete-file", req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error deleting file", err)
		return
	}

	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, req.Id+" "+req.Type+" file queued for deletion", nil)
}
