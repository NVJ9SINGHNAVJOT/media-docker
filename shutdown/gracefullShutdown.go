package shutdown

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// WaitForShutdownSignal gracefully handles shutdown signals and stops the server with a given timeout.
//
// NOTE: It's important to run this function in a separate goroutine to avoid blocking the main thread.
// The timeout is specified in seconds and is the time given for the server to complete any pending requests.
//
// Example: go WaitForShutdownSignal(server, 10)
func WaitForShutdownSignal(srv *http.Server, timeout int) {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)

	// Notify the channel when an interrupt (SIGINT) or termination (SIGTERM) signal is received
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigChan
	log.Info().Msgf("Received signal: %s. Shutting down in %d seconds...", sig, timeout)

	// Create a context with the specified timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer shutdownCancel() // Ensure the cancel function is called to release resources

	// Attempt to gracefully shut down the HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error") // Log if there's an error during shutdown
	} else {
		log.Info().Msg("Server shut down complete.") // Log successful shutdown
	}
}
