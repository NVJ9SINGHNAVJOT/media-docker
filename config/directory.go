package config

import (
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

func CreateDirSetup() {
	// UploadStorage created if not existed
	exist, err := pkg.DirExist(helper.Constants.UploadStorage)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking /" + helper.Constants.UploadStorage + " dir")
		panic(err)
	} else if !exist {
		if err = pkg.CreateDir(helper.Constants.UploadStorage); err != nil {
			log.Error().Str("error", err.Error()).Msg("error while creating /" + helper.Constants.UploadStorage + " dir")
			panic(err)
		}
	}

	/*
		NOTE: with "/media_docker_files" folder "/images" and "/audios" folder is also checked,
		because images and audios are stored directly in folders.
		while each video have their own folder in "/media_docker_files/videos"
	*/
	// images
	exist, err = pkg.DirExist(helper.Constants.MediaStorage + "/images")
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking /" + helper.Constants.MediaStorage + "/images dir")
		panic(err)
	} else if !exist {
		if err = pkg.CreateDir(helper.Constants.MediaStorage + "/images"); err != nil {
			log.Error().Str("error", err.Error()).Msg("error while creating /" + helper.Constants.MediaStorage + "/images dir")
			panic(err)
		}
	}
	// audios
	exist, err = pkg.DirExist(helper.Constants.MediaStorage + "/audios")
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking /" + helper.Constants.MediaStorage + "/audios dir")
		panic(err)
	} else if !exist {
		if err = pkg.CreateDir(helper.Constants.MediaStorage + "/audios"); err != nil {
			log.Error().Str("error", err.Error()).Msg("error while creating /" + helper.Constants.MediaStorage + "/audios dir")
			panic(err)
		}
	}
}
