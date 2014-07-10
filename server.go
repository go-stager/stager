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
		if backend.running {
			backend.proxy.ServeHTTP(writer, request)
		} else {
			writer.WriteHeader(200)
			writer.Write([]byte("The backend you requested is still loading. Check back momentarily."))
		}
	}
}
