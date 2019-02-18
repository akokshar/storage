package modules

import "net/http"

// HTTPHandler interface to be implemented by backend instances
type HTTPHandler interface {
	ServeHTTPRequest(http.ResponseWriter, *http.Request)
	GetRoutePrefix() string
	GetBaseDir() string
}

// FilesDB interface to talk to files database
type FilesDB interface {
	ScanPath(string) int64
	GetPathForID(int64) (string, error)
	GetIDForPath(string) (int64, error)
	GetMetaDataForItemWithID(int64) interface{}
	GetChangesInDirectorySince(id int64, syncAnchor int64, count int) interface{}

	ImportItem(parentID int64, itemPath string) (id int64, err error)
	RemoveItem(id int64) (err error)
}
