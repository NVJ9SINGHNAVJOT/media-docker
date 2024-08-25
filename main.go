package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/routes"
	"github.com/rs/zerolog/log"
)

func main() {
	config.SetUpLogger()

	envs, err := config.ValidateEnvs()
	if err != nil {
		log.Error().Msg("invalid environment variables, " + err.Error())
		panic(err)
	}

	// router
	router := chi.NewRouter()

	// middlewares
	middleware.DefaultMiddlewares(router, envs.AllowedOrigins)

	// Create a route along /media_docker_files that will serve contents from
	// the ./media_docker_files/ folder.
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "media_docker_files"))
	middleware.FileServer(router, "/media_docker_files", filesDir)

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
		log.Error().Msg("error while running server")
		panic(err)
	}
}
