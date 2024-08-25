package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
)

type DeleteFileRequest struct {
	ID   string `json:"id" validate:"required"`
	Type string `json:"type" validate:"required,oneof=image video audio"`
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	var req DeleteFileRequest

	// Parse the JSON request and populate the DeleteFileRequest struct.
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid data", err.Error())
		return
	}

	helper.Response(w, http.StatusOK, req.ID+" file deleted", nil)
}
