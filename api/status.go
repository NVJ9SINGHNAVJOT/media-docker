package api

import (
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
)

type statusRequest struct {
	Id string `json:"id" validate:"required,uuid4"`
}

func Status(w http.ResponseWriter, r *http.Request) {
	var req statusRequest

	// Parse the JSON request and populate the DeleteFileRequest struct.
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid data", err)
		return
	}

	// TODO: To check file by req.id
	helper.Response(w, http.StatusOK, "completed", nil)
}
