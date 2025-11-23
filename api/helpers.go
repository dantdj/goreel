package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dantdj/goreel/storage"
	"github.com/dantdj/goreel/video"
)

// A wrapper for an object to be returned as JSON in a response
type envelope map[string]interface{}

// Takes the destination http.ResponseWriter, the HTTP status code to send,
// the data to encode to JSON, and a header map containing any additional
// HTTP headers to include in the response, and writes the JSON object
// to a given ResponseWriter
func writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// Append a newline to make it easier to view in terminal applications.
	js = append(js, '\n')

	// At this point, we know that we won't encounter any more errors before writing the
	// response, so it's safe to add any headers that we want to include.
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(js); err != nil {
		return err
	}

	return nil
}

func errorResponse(w http.ResponseWriter, status int, message any) {
	env := envelope{"error": message}

	// Write the response using the writeJSON() helper. If this returns an
	// error then log it, and fall back to sending the client an empty response with a
	// 500 Internal Server Error status code.
	err := writeJSON(w, status, env, nil)
	if err != nil {
		slog.Error("Failed to write JSON", slog.String("error", err.Error()))
		w.WriteHeader(500)
	}
}

// Logs the detailed error message, then uses the errorResponse() helper to send
// a 500 Internal Server Error status code and JSON response (containing a generic
// error message) to the client.
func serverErrorResponse(w http.ResponseWriter) {
	message := "the server encountered a problem and could not process your request"
	errorResponse(w, http.StatusInternalServerError, message)
}

// Sends a 404 Not Found status code and JSON response to the client.
func notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	errorResponse(w, http.StatusNotFound, message)
}

// Sends a 405 Method Not Allowed status code and JSON response to the client.
func methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	errorResponse(w, http.StatusMethodNotAllowed, message)
}

func processVideo(videoId string) error {
	storageAccountName := "goreelstorage"
	containerName := "my-go-container"
	storageClient := storage.NewAzureBlobStorage(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"), storageAccountName, containerName)

	videoData, _, _ := storageClient.Retrieve(videoId)
	defer videoData.Close()

	// TODO: Sort this out, probably create the base dir and then
	// the input and output dirs in one method
	baseDir := "./" + videoId + "/"
	inputDir := filepath.Join(baseDir, "input")
	inputPath := filepath.Join(inputDir, videoId)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to make temp directory %s: %w", baseDir, err)
	}

	err := saveToTemp(inputDir, videoId, videoData)
	if err != nil {
		return fmt.Errorf("failed to save video to temp file: %w", err)
	}

	err = video.GenerateM3U8(videoId, inputPath, baseDir)
	if err != nil {
		return fmt.Errorf("failed to generate M3U8 playlist: %w", err)
	}

	playlistFiles, err := getFilePaths(baseDir)
	if err != nil {
		return fmt.Errorf("failed to get file paths: %w", err)
	}

	for _, p := range playlistFiles {
		file, _ := os.Open(p)
		fileInfo, _ := file.Stat()

		storageClient.Upload(file, fileInfo.Size(), file.Name())
	}

	// Remove temporary files
	err = os.RemoveAll(baseDir)
	if err != nil {
		return fmt.Errorf("failed to delete temp files: %w", err)
	}

	err = storageClient.Delete(videoId)
	if err != nil {
		return fmt.Errorf("failed to delete video from storage: %w", err)
	}

	return nil
}

func saveToTemp(fileDir, filename string, videoData io.ReadCloser) error {
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return fmt.Errorf("failed to make input directory %s: %w", fileDir, err)
	}
	filepath := filepath.Join(fileDir, filename)
	outputFile, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create temp input file %s: %w", filepath, err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, videoData)
	if err != nil {
		return fmt.Errorf("failed to copy video data to temp file: %w", err)
	}
	return nil
}

// getFilePaths returns a slice of file paths within a given directory.
func getFilePaths(dirPath string) ([]string, error) {
	var filePaths []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the input directory, we only want the output files
		if d.IsDir() && path == filepath.Join(dirPath, "input") {
			return filepath.SkipDir
		}

		if !d.IsDir() { // Only add files, not directories
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return filePaths, nil
}
