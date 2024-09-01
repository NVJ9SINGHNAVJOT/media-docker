package worker

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"

	"github.com/nvj9singhnavjot/media-docker/config"
	"github.com/rs/zerolog/log"
)

type Media struct {
	command      *exec.Cmd
	wg           *sync.WaitGroup
	commandError *bool
}

var videoChannel chan Media
var videoResolutionChannel chan Media
var imageChannel chan Media
var audioChannel chan Media

func createChannelWorkersPool(channel chan Media, poolSize int) {
	// Pool of workers created of workerSize
	for i := 0; i < poolSize; i++ {
		go func() {
			for media := range channel {
				if err := media.command.Run(); err != nil {
					log.Error().Str("error", err.Error()).Msg(fmt.Sprintf("error executing command, %s", media.command.String()))
					*media.commandError = true
				}
				media.wg.Done()
			}
		}()
	}
}

func SetupChannels() error {
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

	videoChannel = make(chan Media)
	videoResolutionChannel = make(chan Media)
	imageChannel = make(chan Media)
	audioChannel = make(chan Media)

	createChannelWorkersPool(videoChannel, videoWorkers)
	createChannelWorkersPool(videoResolutionChannel, videoWorkers)
	createChannelWorkersPool(imageChannel, videoWorkers)
	createChannelWorkersPool(audioChannel, videoWorkers)

	return nil
}

func CloseChannels() {
	close(videoChannel)
	close(videoResolutionChannel)
	close(imageChannel)
	close(audioChannel)
}

func AddInVideoChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	videoChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}

func AddInVideoResolutionChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	videoResolutionChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}

func AddInImageChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	imageChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}

func AddInAudioChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	audioChannel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}
