package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/dantdj/goreel/utils"
)

var maxRequestBodySize = 500 * 1024 * 1024
var videoProcessingQueueName = "video_processing"

func (app *Application) PingHandler(w http.ResponseWriter, r *http.Request) {
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

func (app *Application) VideoUploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit the overall size of the request body
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxRequestBodySize))

	// Stream the file directly without buffering to disk
	reader, err := r.MultipartReader()
	if err != nil {
		slog.Error("Error creating multipart reader", slog.String("error", err.Error()))
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("Error reading multipart part", slog.String("error", err.Error()))
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		if part.FormName() == "video_file" {
			slog.Info("Starting video upload...")
			blobName, err := utils.GenerateRandomId()
			if err != nil {
				slog.Error("Failed to generate blob name", slog.String("error", err.Error()))
				serverErrorResponse(w)
				return
			}

			blobLocation := app.Storage.Upload(part, blobName)

			slog.Info("Uploaded video", slog.String("video_id", blobName))

			env := envelope{
				"video_id": blobName,
				"location": blobLocation,
			}

			body := []byte(blobName)

			err = app.RabbitClient.Publish(videoProcessingQueueName, body)
			if err != nil {
				slog.Error("Failed to publish message to RabbitMQ", slog.String("error", err.Error()))
				serverErrorResponse(w)
				return
			}

			if err := writeJSON(w, http.StatusOK, env, nil); err != nil {
				slog.Error("Failed to return service info", slog.String("error", err.Error()))
				serverErrorResponse(w)
			}
			return
		}
	}

	http.Error(w, "video_file field not found", http.StatusBadRequest)
}

func (app *Application) RetrieveVideoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("vId")

	videoData, contentLength, contentType := app.Storage.Retrieve(id)
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

func (app *Application) ProcessVideoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("vId")

	if err := app.Processor.Process(id); err != nil {
		slog.Error("Failed to process video", slog.String("video_id", id), slog.String("error", err.Error()))
		serverErrorResponse(w)
	}
}
