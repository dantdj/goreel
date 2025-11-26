package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func routes(app *Application) http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/ping", app.PingHandler)
	router.HandlerFunc(http.MethodPost, "/upload", app.VideoUploadHandler)
	router.HandlerFunc(http.MethodGet, "/download", app.RetrieveVideoHandler)
	router.HandlerFunc(http.MethodGet, "/process", app.ProcessVideoHandler)

	return recoverPanic(enableCORS(router))
}
