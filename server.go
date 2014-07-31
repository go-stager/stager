package stager

import (
	"html/template"
	"net/http"
	"path/filepath"
	"time"
)

func Serve(config *Configuration) {
	backends := NewBackendManager(config)
	backendHandler := BuildBackendHandler(config, backends)
	apiHandler := BuildApiHandler(config, backends)
	muxHandler := BuildStagerRoot(config, backendHandler, apiHandler)
	http.ListenAndServe(config.Listen, muxHandler)
}

// Use BuildStagerRoot to create the root handler for stager.
// The root handler sends API and static requests their specific ways, and
// sends everything else along to the backend handler.
func BuildStagerRoot(config *Configuration, backendHandler http.Handler, apiHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	staticDir := filepath.Clean(filepath.Join(config.ResourceDir, StaticDirName))
	static := "/_stager/static/"
	mux.Handle(static, http.StripPrefix(static, http.FileServer(http.Dir(staticDir))))
	api := "/_stager/api/"
	mux.Handle(api, http.StripPrefix(api, apiHandler))
	mux.Handle("/", backendHandler)
	return mux
}

func BuildBackendHandler(config *Configuration, backends *BackendManager) http.HandlerFunc {
	loading := getLoadingTemplate(config)
	holdFor := config.HoldForDuration()
	return func(writer http.ResponseWriter, request *http.Request) {
		backend, err := backends.Get(request.Host)
		if err != nil {
			simpleTextResponse(
				writer, http.StatusInternalServerError,
				"Got an internal error finding a backend: "+err.Error(),
			)
			return
		}
		switch backend.state {
		case StateNew, StateStarted:
			if holdFor != 0 && request.Method != "GET" {
				// on non-GET requests, wait until we have a ready backend to serve.
				select {
				case <-backend.starter:
					// If we're here, it's because the starter channel signaled. Serve us.
					backend.proxy.ServeHTTP(writer, request)
				case <-time.After(holdFor):
					simpleTextResponse(writer, http.StatusGatewayTimeout, "Backend did not come up within the time limit.")
				}
				return
			}
			render(loading, writer, tdata{"backend": backend})
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

func getLoadingTemplate(config *Configuration) *template.Template {
	fname := filepath.Join(config.ResourceDir, TemplatesDirName, "loading.html")
	return template.Must(template.ParseFiles(fname))
}

type tdata map[string]interface{}
