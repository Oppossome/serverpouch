package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/google/uuid"
)

func main() {
	testCtx, testCtxClose := context.WithCancel(context.Background())
	defer testCtxClose()

	testCtx, err := docker.WithClient(testCtx, nil)
	if err != nil {
		panic(fmt.Sprintf("Unable to start handler: %s\n", err.Error()))
	}

	serverInstance := docker.NewDockerServerInstance(testCtx, &docker.DockerServerInstanceOptions{
		ID:      uuid.Nil,
		Image:   "crccheck/hello-world",
		Volumes: map[string]string{},
		Ports:   map[int]string{80: "8000/tcp"},
		Env:     []string{},
	})

	// Log the output
	go func() {
		statusChan := serverInstance.Events().Status.On()
		terminalOut := serverInstance.Events().TerminalOut.On()

		for {
			select {
			case status, ok := <-statusChan:
				if !ok {
					return
				}
				fmt.Printf("Status: %s\n", status)
			case terminalOut, ok := <-terminalOut:
				if !ok {
					return
				}
				fmt.Printf("TerminalOut: %s\n", terminalOut)
			}
		}
	}()

	// Perform our interactions
	go func() {
		serverInstance.Action(server.ServerInstanceActionStart)
		serverInstance.Events().TerminalIn.Dispatch("Test!")

		serverInstance.Action(server.ServerInstanceActionStop)
	}()

	// Allow ^C to close the program
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	serverInstance.Close()
}
