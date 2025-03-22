package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	httpRepo "oppossome/serverpouch/internal/delivery/http"
	"oppossome/serverpouch/internal/domain/usecases"
	"oppossome/serverpouch/internal/infrastructure/database"
	"oppossome/serverpouch/internal/infrastructure/database/schema"
	"oppossome/serverpouch/internal/infrastructure/docker"

	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	migrate "github.com/rubenv/sql-migrate"
)

// loadEnv loads the environment variables from the .env file, falling back to .env.example if the former is not found
func loadEnv(ctx context.Context) {
	if err := godotenv.Load(".env"); err == nil {
		return
	}

	if err := godotenv.Load(".env.example"); err == nil {
		return
	}

	zerolog.Ctx(ctx).Info().Msg("Failed to find associated .env file")
}

func main() {
	appCtx, appCtxClose := context.WithCancel(context.Background())
	defer appCtxClose()

	// Initialize the logger
	appCtx = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger().WithContext(appCtx)

	loadEnv(appCtx)

	databaseURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		zerolog.Ctx(appCtx).Error().Msg("DATABASE_URL not provided")
		return
	}

	httpURL, ok := os.LookupEnv("HTTP_URL")
	if !ok {
		zerolog.Ctx(appCtx).Info().Msg("HTTP_URL not found")
		return
	}

	// Migrate the database
	_, err := schema.Migrate(appCtx, databaseURL, migrate.Up)
	if err != nil {
		zerolog.Ctx(appCtx).Err(err).Msg("failed to migrate database")
		return
	}

	// Initialize the database
	db, err := database.New(appCtx, databaseURL)
	if err != nil {
		zerolog.Ctx(appCtx).Err(err).Msg("failed to start database")
		return
	}

	appCtx = database.WithDatabase(appCtx, db)

	// Initialize the docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		zerolog.Ctx(appCtx).Err(err).Msg("failed to start docker client")
		return
	}

	appCtx = docker.WithClient(appCtx, dockerClient)

	// Initialize the usecases
	usc, err := usecases.New(appCtx)
	if err != nil {
		zerolog.Ctx(appCtx).Err(err).Msg("failed to start usecases")
		return
	}
	defer usc.Close()

	appCtx = usecases.WithUsecases(appCtx, usc)

	// Initialize our HTTP router
	router, err := httpRepo.New(appCtx)
	if err != nil {
		zerolog.Ctx(appCtx).Err(err).Msg("Failed to initialize our router")
		return
	}

	// Initialize our HTTP server
	httpServer := &http.Server{
		Handler: router,
		Addr:    httpURL,
	}

	go httpServer.ListenAndServe()
	defer httpServer.Shutdown(appCtx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
