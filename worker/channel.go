package worker

import (
	"os/exec"
	"sync"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/rs/zerolog/log"
)

// Flag to control request acceptance
var AcceptingRequests = true

// Create a WaitGroup to track worker goroutines
var wg sync.WaitGroup

// Media struct holds information about the command to be executed,
// a wait group to track completion, and a boolean to indicate if an error occurred.
type Media struct {
	command      *exec.Cmd       // The command to be executed
	wg           *sync.WaitGroup // WaitGroup to manage concurrency and ensure completion of tasks
	commandError *bool           // Pointer to a boolean flag indicating if the command execution failed
}

// Channels to process different media types concurrently
var videoChannel = make(chan Media)           // Channel to handle video processing
var videoResolutionChannel = make(chan Media) // Channel to handle video resolution changes
var imageChannel = make(chan Media)           // Channel to handle image processing
var audioChannel = make(chan Media)           // Channel to handle audio processing

// createChannelWorkersPool creates a pool of worker goroutines that process media tasks from the given channel.
// It runs a specified number of workers (poolSize), each of which listens on the channel for Media commands to execute.
func createChannelWorkersPool(channelName string, channel chan Media, poolSize int) {
	// Pool of workers created based on poolSize
	for i := 0; i < poolSize; i++ {
		workerID := i // Capture workerID for the current worker
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Log worker start
			log.Info().
				Str("channel", channelName).
				Int("worker", workerID).
				Msg("Worker started")

			for media := range channel { // Loop to consume tasks from the channel
				// Execute the media command and handle any errors
				if err := media.command.Run(); err != nil {
					log.Error().
						Str("channel", channelName).
						Int("worker", workerID).
						Err(err).
						Str("command", media.command.String()).
						Msg("Error executing command")
					*media.commandError = true // Set the error flag if command execution fails
				}
				// Signal task completion using WaitGroup
				media.wg.Done()
			}

			// Log worker stop when the channel is closed
			log.Info().
				Str("channel", channelName).
				Int("worker", workerID).
				Msg("Worker stopped")

		}(workerID) // Pass workerID to the goroutine to avoid race condition
	}
}

// SetupChannels initializes channels and worker pools for each media type.
// It reads pool sizes from the environment configuration.
func SetupChannels() {
	// Create worker pools for each channel
	createChannelWorkersPool("videoChannel", videoChannel, config.MDSenvs.VIDEO_WORKER_POOL_SIZE)
	createChannelWorkersPool("videoResolutionChannel", videoResolutionChannel, config.MDSenvs.VIDEO_RESOLUTION_WORKER_POOL_SIZE)
	createChannelWorkersPool("imageChannel", imageChannel, config.MDSenvs.IMAGE_WORKER_POOL_SIZE)
	createChannelWorkersPool("audioChannel", audioChannel, config.MDSenvs.AUDIO_WORKER_POOL_SIZE)
}

// CloseChannels closes all media channels, ensuring no more tasks can be added.
// Set AcceptingRequests to false for new requests
//
// NOTE: We have not used context.Context for graceful shutdown in this implementation
// because this project is Docker-based. When the server is running inside a Docker container,
// all command executions will automatically stop when the container shuts down.
// Docker handles process termination, and all running commands (including exec.Cmd)
// will be killed when the container stops.
//
// However, if this server is started on a local machine (outside of Docker),
// the commands running via exec.Cmd will continue to execute even if the server is closed.
// To handle such cases, you might consider using context.Context to manage the lifecycle
// of commands and ensure they are stopped when the server shuts down.
func CloseChannels() {
	AcceptingRequests = false
	log.Warn().Msg("Stopped accepting new requests for media-docker-server.")
	close(videoChannel)
	close(videoResolutionChannel)
	close(imageChannel)
	close(audioChannel)
	wg.Wait()
}

// AddInVideoChannel adds a Media task to the videoChannel for processing.
func AddInVideoChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	select {
	case videoChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}:
		// Task added successfully
	default:
		log.Warn().
			Str("command", command.String()).
			Msg("Cannot add to videoChannel: channel is closed")
		*commandError = true // Set error flag
		wg.Done()            // Indicate that the task is not being processed
	}
}

// AddInVideoResolutionChannel adds a Media task to the videoResolutionChannel for processing.
func AddInVideoResolutionChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	select {
	case videoResolutionChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}:
		// Task added successfully
	default:
		log.Warn().
			Str("command", command.String()).
			Msg("Cannot add to videoResolutionChannel: channel is closed")
		*commandError = true // Set error flag
		wg.Done()            // Indicate that the task is not being processed
	}
}

// AddInImageChannel adds a Media task to the imageChannel for processing.
func AddInImageChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	select {
	case imageChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}:
		// Task added successfully
	default:
		log.Warn().
			Str("command", command.String()).
			Msg("Cannot add to imageChannel: channel is closed")
		*commandError = true // Set error flag
		wg.Done()            // Indicate that the task is not being processed
	}
}

// AddInAudioChannel adds a Media task to the audioChannel for processing.
func AddInAudioChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	select {
	case audioChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}:
		// Task added successfully
	default:
		log.Warn().
			Str("command", command.String()).
			Msg("Cannot add to audioChannel: channel is closed")
		*commandError = true // Set error flag
		wg.Done()            // Indicate that the task is not being processed
	}
}
