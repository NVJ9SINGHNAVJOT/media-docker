package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/serverKafka"
)

func dirExist(dirPath string) (bool, error) {
	_, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

type DeleteFileRequest struct {
	Id   string `json:"id" validate:"required,uuid4"`
	Type string `json:"type" validate:"required,oneof=image video audio"`
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	var req DeleteFileRequest

	// Parse the JSON request and populate the DeleteFileRequest struct.
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid data", err)
		return
	}

	path := fmt.Sprintf("%s/%ss/%s", helper.Constants.MediaStorage, req.Type, req.Id)

	exist, err := dirExist(path)
	if err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid file for deleting", err)
		return
	}

	if !exist {
		helper.Response(w, http.StatusBadRequest, "file doesn't exist for deleting", nil)
		return
	}

	if err := serverKafka.KafkaProducer.Produce("delete-file", req); err != nil {
		helper.Response(w, http.StatusInternalServerError, "error deleting file", err)
		return
	}

	helper.Response(w, http.StatusOK, req.Id+" "+req.Type+" file deleted", nil)
}
