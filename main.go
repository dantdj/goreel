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

	handler, err := adapter.New(
		adapter.SetDataset(os.Getenv("AXIOM_DATASET")),
		adapter.SetClient(client),
	)
	defer handler.Close()

	//logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := slog.New(handler)
	slog.SetDefault(logger)
	// Start the HTTP server.
	if err := api.Serve(8089); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		return
	}
}
