package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dantdj/goreel/queueing"
	"github.com/dantdj/goreel/storage"
	"github.com/dantdj/goreel/utils"
)

var maxRequestBodySize = 500 * 1024 * 1024
var rabbitClient *queueing.Client

func PingHandler(w http.ResponseWriter, r *http.Request) {
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"serverTimestamp": time.Now().Format(time.RFC3339),
		},
	}

	if err := writeJSON(w, http.StatusOK, env, nil); err != nil {
		slog.Error("Failed to return service info", slog.String("error", err.Error()))
		serverErrorResponse(w)
	}
}

func VideoUploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit the overall size of the request body
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxRequestBodySize))

	err := r.ParseMultipartForm(10 * 1024) // Limit to 10 KB for other form fields
	if err != nil {
		if err.Error() == "http: request body too large" {
			slog.Error("Request body too large, rejecting request", slog.String("error", err.Error()), slog.Int("max_size", maxRequestBodySize/1024/1024), slog.Int64("actual_size", r.ContentLength))
			http.Error(w, fmt.Sprintf("Request body too large. Max allowed is %d MB.", maxRequestBodySize/1024/1024), http.StatusRequestEntityTooLarge)
			return
		}
		slog.Error("Error parsing multipart form", slog.String("error", err.Error()))
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("video_file") // "video_file" is the name of the input in the HTML form
	if err != nil {
		slog.Error("Error retrieving file from form", slog.String("error", err.Error()))
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// TODO: Probably good to do some content type validation

	slog.Info("Starting video upload...")
	storageAccountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	if storageAccountName == "" {
		storageAccountName = "goreelstorage"
	}
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
	if containerName == "" {
		containerName = "videos"
	}
	storageClient := storage.NewAzureBlobStorage(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"), storageAccountName, containerName)
	blobName := utils.GenerateRandomId()

	blobLocation := storageClient.Upload(file, handler.Size, blobName)

	slog.Info("Uploaded video", slog.String("video_id", blobName))

	env := envelope{
		"video_id": blobName,
		"location": blobLocation,
	}

	body := []byte(blobName)

	err = rabbitClient.Publish(videoProcessingQueueName, body)
	if err != nil {
		slog.Error("Failed to publish message to RabbitMQ", slog.String("error", err.Error()))
		serverErrorResponse(w)
		return
	}

	if err := writeJSON(w, http.StatusOK, env, nil); err != nil {
		slog.Error("Failed to return service info", slog.String("error", err.Error()))
		serverErrorResponse(w)
	}
}

func RetrieveVideoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("vId")

	storageAccountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	if storageAccountName == "" {
		storageAccountName = "goreelstorage"
	}
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
	if containerName == "" {
		containerName = "videos"
	}
	storageClient := storage.NewAzureBlobStorage(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"), storageAccountName, containerName)

	videoData, contentLength, contentType := storageClient.Retrieve(id)
	defer videoData.Close()

	slog.Info("Retrieved file", slog.String("file_name", id))

	w.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
	w.Header().Set("Content-Type", contentType)

	_, err := io.Copy(w, videoData)
	if err != nil {
		// At this point, headers have been sent and we can't send an HTTP error status code.
		// The client might receive an incomplete file or a connection reset.
		// Log the error and move on.
		slog.Error("Error streaming file to client", slog.String("file_name", id), slog.String("error", err.Error()))
		return
	}

	slog.Info("Successfully streamed file to client", slog.String("file_name", id))
}

func ProcessVideoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("vId")

	if err := processVideo(id); err != nil {
		slog.Error("Failed to process video", slog.String("video_id", id), slog.String("error", err.Error()))
		serverErrorResponse(w)
	}
}
