package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/internal/media-docker-server/routes"
	"github.com/nvj9singhnavjot/media-docker/kafkahandler"
	mw "github.com/nvj9singhnavjot/media-docker/middleware"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/shutdown"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
)

// cleanUpForServer performs any final cleanup actions before shutdown
func cleanUpForServer() {
	log.Info().Msg("Starting cleanup.")
	if err := kafkahandler.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("Error while closing producer for media-docker-server")
	} else {
		log.Info().Msg("Producer closed for media-docker-server.")
	}

	pkg.CloseDeleteChannels()
	log.Info().Msg("Delete channels closed.")
	time.Sleep(10 * time.Second)
	log.Info().Msg("Cleanup completed.")
}

func main() {
	// Load env file
	err := pkg.LoadEnv(".env.server")
	if err != nil {
		fmt.Println("Error loading env file", err)
		panic(err)
	}

	// environment variables are checked
	err = config.ValidateServerEnv()
	if err != nil {
		fmt.Println("invalid environment variables", err)
		panic(err)
	}

	// logger setup for server
	config.SetUpLogger(config.ServerEnv.ENVIRONMENT)

	pkg.DirExist(helper.Constants.UploadStorage)
	pkg.DirExist(helper.Constants.MediaStorage)

	err = kafkahandler.CheckAllKafkaConnections(config.ServerEnv.KAFKA_BROKERS)
	if err != nil {
		log.Fatal().Err(err).Msg("Kafka connection failed for media-docker-server")
	}

	kafkahandler.InitializeKafkaProducerManager(config.ServerEnv.KAFKA_BROKERS)

	go pkg.DeleteFileWorker()
	go pkg.DeleteDirWorker()

	// Initialize validator
	validator.InitializeValidator()

	// router intialized
	router := chi.NewRouter()

	// NOTE: Adjust throttle middleware value based on the required traffic control
	// all default middlewares initialized
	mw.DefaultMiddlewares(router, config.ServerEnv.ALLOWED_ORIGINS, []string{"POST", "DELETE"}, 10000)

	// server key for accessing server
	router.Use(mw.ServerKey(config.ServerEnv.SERVER_KEY))

	// middlewares for this router
	router.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	router.Use(middleware.AllowContentType("application/json", "multipart/form-data"))
	router.Use(mw.LoggingRequest)

	// all routes for server
	router.Route("/api/v1/uploads", routes.UploadRoutes())
	router.Route("/api/v1/destroys", routes.DestroyRoutes())
	router.Route("/api/v1/connections", routes.ConnectionRoutes())

	// Index handler
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helper.SuccessResponse(w, helper.GetRequestID(r), 200, "server running...", nil)
	})

	// Setup the server with graceful shutdown
	server := &http.Server{
		Addr:    ":" + config.ServerEnv.SERVER_PORT,
		Handler: router,
	}

	// Call graceful shutdown function with a goroutine to avoid blocking the main thread
	go shutdown.WaitForShutdownSignal(server, 60)

	// Start the HTTP server
	log.Info().Msg("media-docker-server is running...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("HTTP server crashed.")
	} else {
		log.Info().Msg("Server gracefully stopped.")
	}

	cleanUpForServer()
	log.Info().Msg("media-docker-server stopped.")
}
