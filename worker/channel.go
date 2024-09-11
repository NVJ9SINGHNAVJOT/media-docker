package worker

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/rs/zerolog/log"
)

// Media struct holds information about the command to be executed,
// a wait group to track completion, and a boolean to indicate if an error occurred.
type Media struct {
	command      *exec.Cmd       // The command to be executed
	wg           *sync.WaitGroup // WaitGroup to manage concurrency and ensure completion of tasks
	commandError *bool           // Pointer to a boolean flag indicating if the command execution failed
}

// Channels to process different media types concurrently
var videoChannel chan Media           // Channel to handle video processing
var videoResolutionChannel chan Media // Channel to handle video resolution changes
var imageChannel chan Media           // Channel to handle image processing
var audioChannel chan Media           // Channel to handle audio processing

// createChannelWorkersPool creates a pool of worker goroutines that process media tasks from the given channel.
// It runs a specified number of workers (poolSize), each of which listens on the channel for Media commands to execute.
func createChannelWorkersPool(channel chan Media, poolSize int) {
	// Pool of workers created based on poolSize
	for i := 0; i < poolSize; i++ {
		go func() {
			for media := range channel { // Loop to consume tasks from the channel
				// Execute the media command and handle any errors
				if err := media.command.Run(); err != nil {
					log.Error().Str("error", err.Error()).Msg(fmt.Sprintf("error executing command, %s", media.command.String()))
					*media.commandError = true // Set the error flag if command execution fails
				}
				// Signal task completion using WaitGroup
				media.wg.Done()
			}
		}()
	}
}

// SetupChannels initializes channels and worker pools for each media type.
// It reads pool sizes from the environment configuration.
func SetupChannels() error {
	// Get worker pool sizes for video, video resolution, image, and audio channels from the config
	videoWorkers, err := strconv.Atoi(config.MDSenvs.VIDEO_WORKER_POOL_SIZE)
	if err != nil {
		return err
	}
	videoResolutionWorkers, err := strconv.Atoi(config.MDSenvs.VIDEO_WORKER_POOL_SIZE)
	if err != nil {
		return err
	}
	imageWorkers, err := strconv.Atoi(config.MDSenvs.VIDEO_WORKER_POOL_SIZE)
	if err != nil {
		return err
	}
	audioWorkers, err := strconv.Atoi(config.MDSenvs.VIDEO_WORKER_POOL_SIZE)
	if err != nil {
		return err
	}

	// Validate worker pool sizes
	if videoWorkers < 1 {
		return fmt.Errorf("minimum size required for video channel workers is 1")
	}
	if videoResolutionWorkers < 1 {
		panic("minimum size required for videoResolution channel workers is 1")
	}
	if imageWorkers < 1 {
		panic("minimum size required for image channel workers is 1")
	}
	if audioWorkers < 1 {
		panic("minimum size required for audio channel workers is 1")
	}

	// Initialize channels for each media type
	videoChannel = make(chan Media)
	videoResolutionChannel = make(chan Media)
	imageChannel = make(chan Media)
	audioChannel = make(chan Media)

	// Create worker pools for each channel
	createChannelWorkersPool(videoChannel, videoWorkers)
	createChannelWorkersPool(videoResolutionChannel, videoResolutionWorkers)
	createChannelWorkersPool(imageChannel, imageWorkers)
	createChannelWorkersPool(audioChannel, audioWorkers)

	return nil
}

// CloseChannels closes all media channels, ensuring no more tasks can be added.
func CloseChannels() {
	close(videoChannel)
	close(videoResolutionChannel)
	close(imageChannel)
	close(audioChannel)
}

// AddInVideoChannel adds a Media task to the videoChannel for processing.
func AddInVideoChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	videoChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}

// AddInVideoResolutionChannel adds a Media task to the videoResolutionChannel for processing.
func AddInVideoResolutionChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	videoResolutionChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}

// AddInImageChannel adds a Media task to the imageChannel for processing.
func AddInImageChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	imageChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}

// AddInAudioChannel adds a Media task to the audioChannel for processing.
func AddInAudioChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	audioChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}
