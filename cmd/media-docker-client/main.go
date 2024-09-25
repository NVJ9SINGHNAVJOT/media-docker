package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

func main() {

	// environment variables are checked
	err := config.ValidateClientEnv()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	// logger setup for server
	config.SetUpLogger(config.ClientEnv.ENVIRONMENT)

	exist, err := pkg.DirExist(helper.Constants.MediaStorage)
	if err != nil {
		log.Fatal().Err(err).Msg("error while checking /" + helper.Constants.MediaStorage + " dir")
	} else if !exist {
		log.Fatal().Msg(helper.Constants.MediaStorage + " dir does not exist")
	}

	// router intialized
	router := chi.NewRouter()

	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.ClientEnv.ALLOWED_ORIGINS, []string{"GET"}, 5000)

	// middlewares for this router
	router.Use(httprate.LimitByIP(10, 1*time.Minute))

	/*
		Create a route along "/media_docker_files" that will serve contents from
		the ./media_docker_files/folder.
	*/
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, helper.Constants.MediaStorage))
	mw.FileServer(router, "/"+helper.Constants.MediaStorage, filesDir)

	// index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	// port initialized
	port := ":" + config.ClientEnv.CLIENT_PORT
	log.Info().Msg("media-docker-client server running...")

	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Fatal().Err(err).Str("error", err.Error()).Msg("error while running server")
	}
}
