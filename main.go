package main

import (
	"log/slog"
	"os"

	"github.com/dantdj/goreel/api"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", slog.String("error", err.Error()))
		return
	}

	// Start the HTTP server.
	if err := api.Serve(8089); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		return
	}
}
