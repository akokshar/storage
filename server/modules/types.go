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
	GetMetaDataForChildrenOfID(id int64, offset int, count int) interface{}
}
