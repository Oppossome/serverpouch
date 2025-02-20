package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oppossome/serverpouch/internal/domain/server"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/google/uuid"
)

func main() {
	testCtx, testCtxClose := context.WithCancel(context.Background())
	defer testCtxClose()

	serverInstance, err := docker.New(testCtx, &docker.DockerServerInstanceOptions{
		ID:      uuid.Nil,
		Image:   "crccheck/hello-world",
		Volumes: map[string]string{},
		Ports:   map[int]string{80: "8000/tcp"},
		Env:     []string{},
	})
	if err != nil {
		panicMsg := fmt.Sprintf("Unable to start handler: %s", err.Error())
		panic(panicMsg)
	}

	// Log the output
	go func() {
		statusChan := serverInstance.Events().Status.On()
		terminalOut := serverInstance.Events().TerminalOut.On()

		for {
			select {
			case status := <-statusChan:
				fmt.Printf("Status: %s\n", status)
			case terminalOut := <-terminalOut:
				fmt.Printf("TerminalOut: %s\n", terminalOut)
			}
		}
	}()

	// Perform our interactions
	go func() {
		time.Sleep(time.Second * 8)

		serverInstance.Action(server.ServerInstanceActionStart)
	}()

	// Allow ^C to close the program
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
