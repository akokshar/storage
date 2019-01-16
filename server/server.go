package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/akokshar/storage/server/handlers"
	"github.com/akokshar/storage/server/handlers/meta"
	"github.com/akokshar/storage/server/handlers/photos"
	"github.com/akokshar/storage/server/handlers/store"
)

type application struct {
	handlers []handlers.Handler
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, h := range app.handlers {
		if !strings.HasPrefix(r.URL.Path, h.GetRoutePrefix()) {
			continue
		}
		localFilePath := path.Join(h.GetBaseDir(), strings.TrimPrefix(r.URL.Path, h.GetRoutePrefix()))
		if !strings.HasPrefix(localFilePath, h.GetBaseDir()) {
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

func (app *application) registerHandler(h handlers.Handler) {
	dir, err := os.Stat(h.GetBaseDir())

	switch {
	case os.IsNotExist(err):
		err = os.MkdirAll(h.GetBaseDir(), os.ModePerm)
		break
	case err != nil:
		break
	case dir.IsDir():
		break
	default:
		log.Fatalf("Handler '%s' initialization fail. '%s' is not a directory.", h.GetRoutePrefix(), h.GetBaseDir())
	}

	if err != nil {
		log.Fatalf("Handler '%s' failed with error '%s'", h.GetRoutePrefix(), err.Error())
	}

	app.handlers = append(app.handlers, h)
	log.Printf("Handler '%s' initialized", h.GetRoutePrefix())
}

// CreateApplication initializes new storage server application
func CreateApplication(basedir string) http.Handler {
	app := &application{
		handlers: make([]handlers.Handler, 0, 3),
	}

	app.registerHandler(meta.New("/meta", basedir))
	app.registerHandler(store.New("/store", path.Join(basedir, "store")))
	app.registerHandler(photos.New("/photos", path.Join(basedir, "photos")))

	return app
}
