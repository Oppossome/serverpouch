package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"oppossome/serverpouch/internal/domain/usecases"
	"oppossome/serverpouch/internal/infrastructure/database"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
)

// loadEnv loads the environment variables from the .env file, falling back to .env.example if the former is not found
func loadEnv() {
	if err := godotenv.Load(".env"); err == nil {
		return
	}

	if err := godotenv.Load(".env.example"); err == nil {
		return
	}

	fmt.Println("Failed to find .env file")
}

func main() {
	loadEnv()

	databaseURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		fmt.Println("DATABASE_URL not provided!")
		return
	}

	appCtx, appCtxClose := context.WithCancel(context.Background())
	defer appCtxClose()

	// Initialize the database
	db, err := database.New(appCtx, databaseURL)
	if err != nil {
		fmt.Printf("Unable to start database %s", err)
		return
	}

	appCtx = database.WithDatabase(appCtx, db)

	// Initialize the docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("Unable to start docker client %s", err)
		return
	}

	appCtx = docker.WithClient(appCtx, dockerClient)

	// Initialize the usecases
	usecases, err := usecases.New(appCtx)
	if err != nil {
		fmt.Printf("Unable to start usecases %s", err)
		return
	}
	defer usecases.Close()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
