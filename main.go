package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/routes"
	"github.com/rs/zerolog/log"
)

func main() {
	// environment variables are checked
	envs, err := config.ValidateEnvs()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	// logger setup for server
	config.SetUpLogger(envs.Environment)

	// UploadStorage created if not existed
	exist, err := pkg.DirExist(helper.Constants.UploadStorage)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking /" + helper.Constants.UploadStorage + " dir")
		panic(err)
	} else if !exist {
		pkg.CreateDir(helper.Constants.UploadStorage)
	}

	/*
		NOTE: with "/media_docker_files" folder "/images" and "/audios" folder is also checked,
		because images and audios are stored directly in folders.
		while each video have their own folder
	*/
	// images
	exist, err = pkg.DirExist(helper.Constants.MediaStorage + "/images")
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking /" + helper.Constants.MediaStorage + "/images dir")
		panic(err)
	} else if !exist {
		pkg.CreateDir(helper.Constants.MediaStorage + "/images")
	}
	// audios
	exist, err = pkg.DirExist(helper.Constants.MediaStorage + "/audios")
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking /" + helper.Constants.MediaStorage + "/audios dir")
		panic(err)
	} else if !exist {
		pkg.CreateDir(helper.Constants.MediaStorage + "/audios")
	}

	// router
	router := chi.NewRouter()

	// all default middlewares initialized
	middleware.DefaultMiddlewares(envs.AllowedOrigins, router)
	// server key for accessing server
	router.Use(middleware.ServerKey(envs.ServerKey))

	/*
		Create a route along "/media_docker_files" that will serve contents from
		the ./media_docker_files/folder.
	*/
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, helper.Constants.MediaStorage))
	middleware.FileServer(router, "/"+helper.Constants.MediaStorage, filesDir)

	// all routes for server
	router.Route("/api/v1/uploads", routes.UploadRoutes())
	router.Route("/api/v1/destroys", routes.DestroyRoutes())

	// index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	// port initialized
	port := ":" + envs.Port
	log.Info().Msg("server running...")

	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while running server")
		panic(err)
	}
}
