package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/routes"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/worker"
	"github.com/rs/zerolog/log"
)

func main() {

	// environment variables are checked
	err := config.ValidateMDSenvs()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	// logger setup for server
	config.SetUpLogger(config.MDSenvs.ENVIRONMENT)

	// check dir setup for server
	config.CreateDirSetup()

	// set channel for command execution
	workerSize, err := strconv.Atoi(config.MDSenvs.WORKER_POOL_SIZE)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while getting worker size for pool")
		panic(err)
	}
	worker.SetupChannel(workerSize)
	defer worker.CloseChannel()

	// HACK: server can use max 1 core only
	runtime.GOMAXPROCS(1)

	// router intialized
	router := chi.NewRouter()

	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.MDSenvs.ALLOWED_ORIGINS_SERVER, []string{"POST", "DELETE"}, 1000)
	// server key for accessing server
	router.Use(mw.ServerKey(config.MDSenvs.SERVER_KEY))
	// middlewares for this router
	router.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	router.Use(middleware.AllowContentType("application/json", "multipart/form-data"))
	router.Use(httprate.LimitByIP(10, 1*time.Minute))

	// all routes for server
	router.Route("/api/v1/uploads", routes.UploadRoutes())
	router.Route("/api/v1/destroys", routes.DestroyRoutes())

	// index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.Response(w, 200, "server running...", nil)
	})

	// port initialized
	port := ":" + config.MDSenvs.SERVER_PORT
	log.Info().Msg("server running...")

	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while running server")
		panic(err)
	}
}
