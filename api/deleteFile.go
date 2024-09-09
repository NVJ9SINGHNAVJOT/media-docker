package api

import (
	"fmt"
	"net/http"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
)

type deleteFileRequest struct {
	Id   string `json:"id" validate:"required,uuid4"`
	Type string `json:"type" validate:"required,oneof=image video audio"`
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	var req deleteFileRequest

	// Parse the JSON request and populate the DeleteFileRequest struct.
	if err := helper.ValidateRequest(r, &req); err != nil {
		helper.Response(w, http.StatusBadRequest, "invalid data", err.Error())
		return
	}

	path := fmt.Sprintf("%s/%ss/%s", helper.Constants.MediaStorage, req.Type, req.Id)

	/*
		NOTE: no error is passed in response as for file or dir deleting logging is
		done inside deleting function only.
	*/
	if req.Type == "image" {
		err := pkg.DeleteFile(path + ".jpeg")
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "invalid image file for deleting", nil)
			return
		}
	} else if req.Type == "audio" {
		err := pkg.DeleteFile(path + ".mp3")
		if err != nil {
			helper.Response(w, http.StatusBadRequest, "invalid audio file for deleting", nil)
			return
		}
	} else {
		exist, err := pkg.DirExist(path)
		if err != nil || !exist {
			helper.Response(w, http.StatusBadRequest, "invalid video file for deleting", nil)
			return
		}
		pkg.DeleteDir(path)
	}

	helper.Response(w, http.StatusOK, req.Id+" file deleted", nil)
}
