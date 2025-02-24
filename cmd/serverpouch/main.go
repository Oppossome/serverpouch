package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/docker/docker/client"
)

func main() {
	appCtx, appCtxClose := context.WithCancel(context.Background())

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("Unable to start docker client %s", err)
		return
	}

	appCtx = docker.WithClient(appCtx, dockerClient)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	appCtxClose()
}
