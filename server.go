package stager

import (
	"net/http"
)

func Serve(config *Configuration) {
	backends := newBackendManager(config)
	handler := buildHandler(backends)
	http.ListenAndServe(config.Listen, handler)
}

func buildHandler(backends *backendManager) http.HandlerFunc {

	return func(writer http.ResponseWriter, request *http.Request) {
		backend := backends.get(request.Host)
		switch backend.state {
		case StateNew:
			writer.WriteHeader(200)
			writer.Write([]byte("The backend you requested is being built. Check back momentarily."))
		case StateStarted:
			writer.WriteHeader(200)
			writer.Write([]byte("The backend you requested is starting up. Check back momentarily."))
		case StateRunning:
			backend.proxy.ServeHTTP(writer, request)
		case StateFinished:
			writer.WriteHeader(200)
			writer.Write([]byte("The backend you requested is starting up. Check back momentarily."))
		}
	}
}
