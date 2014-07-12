package stager

import (
	"net/http"
	"time"
)

func Serve(config *Configuration) {
	backends := newBackendManager(config)
	handler := buildHandler(backends)
	http.ListenAndServe(config.Listen, handler)
}

func buildHandler(backends *backendManager) http.HandlerFunc {

	return func(writer http.ResponseWriter, request *http.Request) {
		backend, err := backends.get(request.Host)
		if err != nil {
			simpleTextResponse(
				writer, 500,
				"Got an internal error finding a backend: "+err.Error(),
			)
			return
		}
		switch backend.state {
		case StateNew:
			simpleTextResponse(writer, 200, "The backend you requested is being built. Check back momentarily.")
		case StateStarted:
			simpleTextResponse(writer, 200, "The backend you requested is starting up. Check back momentarily.")
		case StateRunning:
			backend.LastReq = time.Now()
			backend.proxy.ServeHTTP(writer, request)
		case StateFinished:
			simpleTextResponse(writer, 200, "The backend you requested has finished. will be cleaning up.")
		case StateErrored:
			simpleTextResponse(writer, 200, "The backend errored after startup. Check your log for reason code.")
		}
	}
}
