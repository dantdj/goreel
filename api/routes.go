package api

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/dantdj/goreel/queueing"
	"github.com/julienschmidt/httprouter"
)

var videoProcessingQueueName = "video_processing"

func routes() http.Handler {
	router := httprouter.New()

	rabbitUrl := os.Getenv("RABBITMQ_URL")
	var err error
	rabbitClient, err = queueing.NewRabbitClient(rabbitUrl)
	if err != nil {
		panic("couldn't set up RabbitMQ client")
	}

	rabbitClient.StartConsumer(videoProcessingQueueName, func(message []byte) error {
		processVideo(string(message))
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
