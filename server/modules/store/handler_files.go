package store

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/akokshar/storage/server/modules"
)

const (
	optCmd                = "cmd"
	optCmdCreateDir       = "createDir"
	optCmdListChanges     = "list"
	optCmdInfo            = "info"
	optCmdSyncStatus      = "syncStatus"
	optID                 = "id"
	optParentID           = "parentId"
	optName               = "name"
	optAnchor             = "anchor"
	optAnchorDefaultValue = 0
	optCount              = "count"
	optCountDefaultValue  = 10
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
	opts, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var id int64
	rawID, err := strconv.Atoi(opts.Get(optID))
	if err != nil {
		if opts.Get(optID) == "NSFileProviderRootContainerItemIdentifier" {
			id = s.rootID
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		id = int64(rawID)
	}

	idPath, err := s.filesDB.GetPathForID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if !strings.HasPrefix(idPath, r.Header.Get("X-Local-Filepath")) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	switch opts.Get(optCmd) {
	case optCmdInfo:
		metaData := s.filesDB.GetMetaDataForItemWithID(id)
		metaJSON, _ := json.MarshalIndent(metaData, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(metaJSON)
		break
	case optCmdListChanges:
		var syncAnchor, count int
		if syncAnchor, err = strconv.Atoi(opts.Get(optAnchor)); err != nil {
			syncAnchor = optAnchorDefaultValue
		}
		if count, err = strconv.Atoi(opts.Get(optCount)); err != nil {
			count = optCountDefaultValue
		}

		metaData := s.filesDB.GetChangesInDirectorySince(id, int64(syncAnchor), count)
		metaJSON, _ := json.MarshalIndent(metaData, "", "  ")

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")

		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()
		gzipWriter.Write(metaJSON)
		break
	default:
		http.ServeFile(w, r, idPath)
	}

}

func (s *store) createFile(w http.ResponseWriter, r *http.Request) {
	opts, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var parentID int64
	rawParentID, err := strconv.Atoi(opts.Get(optParentID))
	if err != nil {
		if opts.Get(optParentID) == "NSFileProviderRootContainerItemIdentifier" {
			parentID = s.rootID
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		parentID = int64(rawParentID)
	}

	name := opts.Get(optName)
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	parentPath, err := s.filesDB.GetPathForID(parentID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if !strings.HasPrefix(parentPath, r.Header.Get("X-Local-Filepath")) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	id, err := s.filesDB.CreateItemPlaceholder(parentID, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filePath, err := s.filesDB.GetPathForID(id)

	if opts.Get(optCmd) == optCmdCreateDir {
		// the directory might exist. Do not rase an error, just consume existing directory.
		os.Mkdir(filePath, 0755)
		err = s.filesDB.ImportItem(id, filePath)
		if err != nil {
			s.filesDB.DeleteItemPlaceholder(id)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		f, err := os.Create(filePath)
		if err != nil {
			s.filesDB.DeleteItemPlaceholder(id)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// TODO: http://www.grid.net.ru/nginx/resumable_uploads.en.html
		_, err = io.Copy(f, r.Body)
		if err != nil {
			s.filesDB.DeleteItemPlaceholder(id)
			os.Remove(filePath)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = s.filesDB.ImportItem(id, filePath)
		if err != nil {
			log.Printf("%v", err)
			s.filesDB.DeleteItemPlaceholder(id)
			os.Remove(filePath)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	metaData := s.filesDB.GetMetaDataForItemWithID(id)
	metaJSON, _ := json.MarshalIndent(metaData, "", "  ")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(metaJSON)
}

func (s *store) deleteFile(w http.ResponseWriter, r *http.Request) {
	opts, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var id int64
	rawID, err := strconv.Atoi(opts.Get(optID))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id = int64(rawID)

	if id == s.rootID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	idPath, err := s.filesDB.GetPathForID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if !strings.HasPrefix(idPath, r.Header.Get("X-Local-Filepath")) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = s.filesDB.RemoveItem(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := os.Remove(idPath); err != nil {
		log.Printf("Failed to delete '%s' due to '%s'", idPath, err.Error())
	}

	w.WriteHeader(http.StatusOK)
}
