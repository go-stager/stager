package stager

import (
	"net/http"
)

func BuildApiHandler(config *Configuration, backends *BackendManager) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// We don't have an extensive API, so no need to over-build this just yet.
		if request.URL.Path == "ready" {
			backend, err := backends.Get(request.Host)
			if err != nil {
				simpleTextResponse(
					writer, http.StatusInternalServerError,
					"Got an internal error finding a backend: "+err.Error(),
				)
				return
			}
			if backend.state == StateRunning {
				simpleTextResponse(writer, http.StatusOK, "true")
			} else {
				simpleTextResponse(writer, http.StatusOK, "false")
			}
		} else {
			simpleTextResponse(writer, http.StatusNotFound, "Stager API method "+request.URL.Path+"not found.")
		}
	}
}
