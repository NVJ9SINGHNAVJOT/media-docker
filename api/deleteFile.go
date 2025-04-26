package api

import (
	"fmt"
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

type DeleteFileRequest struct {
	Id   string `json:"id" validate:"required,uuid4"`
	Type string `json:"type" validate:"required,oneof=image video audio"`
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	var req DeleteFileRequest

	// Parse the JSON request and populate the DeleteFileRequest struct.
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid data", err)
		return
	}

	path := fmt.Sprintf("%s/%ss/%s", helper.Constants.MediaStorage, req.Type, req.Id)

	exist, err := pkg.DirOrFileExist(path)
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

	helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, req.Id+" "+req.Type+" file deleted", nil)
}
