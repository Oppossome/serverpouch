package main

import (
	"context"
	"fmt"
	"io/fs"
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

	handler, err := docker.New(testCtx, &docker.DockerOptions{
		ID:      uuid.Nil,
		Image:   "crccheck/hello-world",
		Volumes: map[fs.DirEntry]string{},
		Ports:   map[int]string{80: "8000/tcp"},
		Env:     []string{},
	})
	if err != nil {
		panicMsg := fmt.Sprintf("Unable to start handler: %s", err.Error())
		panic(panicMsg)
	}

	// Log the output
	go func() {
		statusChan := handler.Events().Status.On()
		terminalOut := handler.Events().TerminalOut.On()

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

		handler.Action(server.HandlerActionStart)
	}()

	// Allow ^C to close the program
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
