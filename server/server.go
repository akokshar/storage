package server

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/akokshar/storage/server/backend"
	"github.com/akokshar/storage/server/backend/meta"
	"github.com/akokshar/storage/server/backend/store"
)

type application struct {
	basedir  string
	handlers []backend.Handler
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, h := range app.handlers {
		if !strings.HasPrefix(r.URL.Path, h.GetRoutePrefix()) {
			continue
		}
		localFilePath := path.Join(app.basedir, strings.TrimPrefix(r.URL.Path, h.GetRoutePrefix()))
		if !strings.HasPrefix(localFilePath, app.basedir) {
			log.Print(fmt.Sprintf("Access to '%s' denied", localFilePath))
			w.WriteHeader(http.StatusForbidden)
			return
		}
		//r.URL.Path = localFilePath
		r.Header.Add("X-Local-Filepath", localFilePath)
		h.ServeHTTPRequest(w, r)
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

// CreateApplication initializes new storage server application
func CreateApplication(basedir string) http.Handler {
	app := &application{
		basedir:  basedir,
		handlers: make([]backend.Handler, 2),
	}
	app.handlers[0] = meta.New()
	app.handlers[1] = store.New()

	return app
}
