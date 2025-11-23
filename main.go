package main

import (
	"log/slog"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/dantdj/goreel/api"
	"github.com/joho/godotenv"

	adapter "github.com/axiomhq/axiom-go/adapters/slog"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", slog.String("error", err.Error()))
		return
	}

	client, err := axiom.NewClient(
		axiom.SetPersonalTokenConfig(os.Getenv("AXIOM_TOKEN"), os.Getenv("AXIOM_ORG_ID")),
	)
	if err != nil {
		slog.Error("Error creating axiom client", slog.String("error", err.Error()))
		return
	}

	handler, err := adapter.New(
		adapter.SetDataset(os.Getenv("AXIOM_DATASET")),
		adapter.SetClient(client),
	)
	if err != nil {
		slog.Error("Error creating slog handler", slog.String("error", err.Error()))
		return
	}
	defer handler.Close()

	var logger *slog.Logger
	// Allow use of JSON output in local environment,
	// but default to Axiom in production
	if os.Getenv("GOREEL_LOCAL") == "true" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(handler)
	}
	slog.SetDefault(logger)

	// Start the HTTP server.
	if err := api.Serve(8089); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		return
	}
}
