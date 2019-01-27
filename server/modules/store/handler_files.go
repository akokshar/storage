package store

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/akokshar/storage/server/modules"
)

type store struct {
	routePrefix string
	basedir     string
	filesDB     modules.FilesDB
	rootID      int64
}

// New initializes backend to server files
func New(db modules.FilesDB, prefix string, basedir string) modules.HTTPHandler {
	return &store{
		routePrefix: prefix,
		basedir:     basedir,
		filesDB:     db,
		rootID:      db.ScanPath(basedir),
	}
}

func (s *store) GetRoutePrefix() string {
	return s.routePrefix
}

func (s *store) GetBaseDir() string {
	return s.basedir
}

func (s *store) ServeHTTPRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getFile(w, r)
	case http.MethodPost:
		s.createFile(w, r)
	case http.MethodDelete:
		s.deleteFile(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *store) getFile(w http.ResponseWriter, r *http.Request) {
	localFilePath := r.Header.Get("X-Local-Filepath")
	http.ServeFile(w, r, localFilePath)
}

func (s *store) createFile(w http.ResponseWriter, r *http.Request) {
	localFilePath := r.Header.Get("X-Local-Filepath")

	if _, err := os.Stat(localFilePath); err == nil {
		w.WriteHeader(http.StatusConflict)
		return
	}

	f, err := os.Create(localFilePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// TODO: http://www.grid.net.ru/nginx/resumable_uploads.en.html
	_, err = io.Copy(f, r.Body)

	w.WriteHeader(http.StatusCreated)
}

func (s *store) deleteFile(w http.ResponseWriter, r *http.Request) {
	localFilePath := r.Header.Get("X-Local-Filepath")

	if _, err := os.Stat(localFilePath); err == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := os.Remove(localFilePath); err != nil {
		log.Printf("Failed to delete '%s' due to '%s'", localFilePath, err.Error())
		w.WriteHeader(http.StatusNotModified)
	}
}
