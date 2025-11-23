package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
