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
	envs, err := config.ValidateEnvs()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	config.SetUpLogger(envs.Environment)

	exist, err := pkg.DirExist(helper.Constants.UploadStorage)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking " + helper.Constants.UploadStorage + " dir")
		panic(err)
	} else if !exist {
		pkg.CreateDir(helper.Constants.UploadStorage)
	}

	exist, err = pkg.DirExist(helper.Constants.MediaStorage)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while checking " + helper.Constants.MediaStorage + " dir")
		panic(err)
	} else if !exist {
		pkg.CreateDir(helper.Constants.MediaStorage)
	}

	// router
	router := chi.NewRouter()

	// middlewares
	middleware.DefaultMiddlewares(router, envs.AllowedOrigins)

	// Create a route along /media_docker_files that will serve contents from
	// the ./media_docker_files/ folder.
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, helper.Constants.MediaStorage))
	middleware.FileServer(router, "/"+helper.Constants.MediaStorage, filesDir)

	// routes
	router.Route("/api/v1/uploads", routes.UploadRoutes())

	// index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	port := ":" + envs.Port

	log.Info().Msg("server running...")

	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while running server")
		panic(err)
	}
}
