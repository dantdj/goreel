package api

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/dantdj/goreel/queueing"
	"github.com/dantdj/goreel/storage"
	"github.com/dantdj/goreel/video"
	"github.com/julienschmidt/httprouter"
)

var videoProcessingQueueName = "video_processing"

func routes() http.Handler {
	router := httprouter.New()

	rabbitUrl := os.Getenv("RABBITMQ_URL")
	var err error
	rabbitClient, err = queueing.NewRabbitClient(rabbitUrl)
	if err != nil {
		slog.Error("Failed to create RabbitMQ client", slog.String("error", err.Error()))
		panic("couldn't set up RabbitMQ client")
	}

	rabbitClient.StartConsumer(videoProcessingQueueName, func(message []byte) error {
		storageAccountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
		if storageAccountName == "" {
			storageAccountName = "goreelstorage"
		}
		containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
		if containerName == "" {
			containerName = "videos"
		}
		storageClient := storage.NewAzureBlobStorage(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"), storageAccountName, containerName)
		processor := video.NewProcessor(storageClient)

		slog.Info("Received a message", slog.String("body", string(message)))
		if err := processor.Process(string(message)); err != nil {
			slog.Error("Error processing video", slog.String("error", err.Error()))
		}
		return nil
	})

	slog.Info("RabbitMQ consumer started", slog.String("queue", videoProcessingQueueName))

	router.NotFound = http.HandlerFunc(notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/ping", PingHandler)
	router.HandlerFunc(http.MethodPost, "/upload", VideoUploadHandler)
	router.HandlerFunc(http.MethodGet, "/download", RetrieveVideoHandler)
	router.HandlerFunc(http.MethodGet, "/process", ProcessVideoHandler)

	return recoverPanic(router)
}
