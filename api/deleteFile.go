package api

import (
	"fmt"
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
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
		helper.Response(w, http.StatusBadRequest, "invalid data", err.Error())
		return
	}

	path := fmt.Sprintf("%s/%ss/%s", helper.Constants.MediaStorage, req.Type, req.Id)

	exist, err := pkg.DirExist(path)
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid file for deleting", err.Error())
		return
	}

	if !exist {
		helper.Response(w, http.StatusBadRequest, "file doesn't exist for deleting", nil)
		return
	}

	if err := serverKafka.KafkaProducer.Produce("deleteFile", req); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error deleting file", err.Error())
		return
	}

	helper.Response(w, http.StatusOK, req.Id+" file deleted", nil)
}
