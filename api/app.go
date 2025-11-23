package api

import (
	"log/slog"
	"os"

	"github.com/dantdj/goreel/queueing"
	"github.com/dantdj/goreel/storage"
	"github.com/dantdj/goreel/video"
)

type Application struct {
	Storage      storage.Service
	RabbitClient *queueing.Client
	Processor    *video.Processor
}

func NewApplication() *Application {
	// Storage account setup
	storageAccountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	if storageAccountName == "" {
		storageAccountName = "goreelstorage"
	}
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
	if containerName == "" {
		containerName = "videos"
	}
	storageClient := storage.NewAzureBlobStorage(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"), storageAccountName, containerName)

	// RabbitMQ setup
	rabbitUrl := os.Getenv("RABBITMQ_URL")
	rabbitClient, err := queueing.NewRabbitClient(rabbitUrl)
	if err != nil {
		slog.Error("Failed to create RabbitMQ client", slog.String("error", err.Error()))
		panic("couldn't set up RabbitMQ client")
	}

	// Video processor setup
	processor := video.NewProcessor(storageClient)

	return &Application{
		Storage:      storageClient,
		RabbitClient: rabbitClient,
		Processor:    processor,
	}
}

// Starts a consumer that processes video processing requests from RabbitMQ.
func (app *Application) StartConsumers() {
	videoProcessingQueueName := "video_processing"
	app.RabbitClient.StartConsumer(videoProcessingQueueName, func(message []byte) error {
		slog.Info("Received a message", slog.String("body", string(message)))
		if err := app.Processor.Process(string(message)); err != nil {
			slog.Error("Error processing video", slog.String("error", err.Error()))
		}
		return nil
	})
	slog.Info("RabbitMQ consumer started", slog.String("queue", videoProcessingQueueName))
}
