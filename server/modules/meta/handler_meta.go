package meta

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/akokshar/storage/server/modules"
)

const (
	enumerateOptName   = "enumerate"
	countOptName       = "count"
	countDefaultValue  = 10
	offsetOptName      = "offset"
	offsetDefaultValue = 0
)

type meta struct {
	routePrefix string
	basedir     string
	filesDB     modules.FilesDB
}

// New initializes backend to serve metadata
func New(db modules.FilesDB, prefix string, basedir string) modules.HTTPHandler {
	return &meta{
		routePrefix: prefix,
		basedir:     basedir,
		filesDB:     db,
	}
}

func (m *meta) GetRoutePrefix() string {
	return m.routePrefix
}

func (m *meta) GetBaseDir() string {
	return m.basedir
}

func (m *meta) ServeHTTPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	opts, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var metaData interface{}
	var id int64

	optsID, err := strconv.Atoi(opts.Get("id"))
	if err != nil {
		localFilePath := r.Header.Get("X-Local-Filepath")
		if id, err = m.filesDB.GetIDForPath(localFilePath); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		id = int64(optsID)
	}

	if opts.Get("enumerate") != "" {
		var offset, count int
		if offset, err = strconv.Atoi(opts.Get(offsetOptName)); err != nil {
			offset = offsetDefaultValue
		}
		if count, err = strconv.Atoi(opts.Get(countOptName)); err != nil {
			count = countDefaultValue
		}
		metaData = m.filesDB.GetMetaDataForChildrenOfID(id, offset, count)
	} else {
		metaData = m.filesDB.GetMetaDataForItemWithID(int64(id))
	}

	metaJSON, _ := json.MarshalIndent(metaData, "", "  ")
	w.Header().Add("Content-Type", "application/json")
	w.Write(metaJSON)
}
