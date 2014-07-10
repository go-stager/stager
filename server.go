package stager

import (
	"net/http"
)

func Serve(config *Configuration) {
	handler := buildHandler(config)
	http.ListenAndServe(config.Listen, handler)
}

func buildHandler(config *Configuration) http.HandlerFunc {
	backends := newBackendManager(config)

	return func(writer http.ResponseWriter, request *http.Request) {
		backends.get(request.Host).proxy.ServeHTTP(writer, request)
	}
}
