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
	envErr := godotenv.Load()

	var handler slog.Handler
	// We only care about creating the Axiom handler if we're not in local mode,
	// as we shouldn't be logging there by default
	if os.Getenv("GOREEL_LOCAL") != "true" {
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
	}

	var logger *slog.Logger
	// Allow use of JSON output in local environment,
	// but default to Axiom in production
	if os.Getenv("GOREEL_LOCAL") == "true" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(handler)
	}
	slog.SetDefault(logger)

	if envErr != nil {
		// Now that we've set up logging, we can log the
		// information about the missing .env file.
		// We might be in prod with no .env file, so
		// log a message but continue
		slog.Info("No .env file found, relying on environment variables")
	}

	// Start the HTTP server.
	if err := api.Serve(8089); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		return
	}
}
