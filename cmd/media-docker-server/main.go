package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/routes"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
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
	config.SetUpLogger(config.MDSenvs.Environment)

	// check dir setup for server
	config.CreateDirSetup()

	// router intialized
	router := chi.NewRouter()

	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.MDSenvs.AllowedOrigins, []string{"POST", "DELETE"}, 1000)
	// server key for accessing server
	router.Use(mw.ServerKey(config.MDSenvs.ServerKey))
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
	port := ":" + config.MDSenvs.Port
	log.Info().Msg("server running...")

	err = http.ListenAndServe(port, router)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("error while running server")
		panic(err)
	}
}
