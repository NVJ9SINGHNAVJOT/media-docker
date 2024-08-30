package worker

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/rs/zerolog/log"
)

type Media struct {
	command      *exec.Cmd
	wg           *sync.WaitGroup
	commandError *bool
}

var channel chan Media

func SetupChannel(workerSize int) {
	if workerSize < 1 {
		panic("minimum size required for channel workers is 1")
	}

	channel = make(chan Media)

	// Pool of workers created of workerSize
	for i := 0; i < workerSize; i++ {
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

func CloseChannel() {
	close(channel)
}

func AddInChannel(command *exec.Cmd, wg *sync.WaitGroup, commandError *bool) {
	channel <- Media{
		command:      command,
		wg:           wg,
		commandError: commandError,
	}
}
