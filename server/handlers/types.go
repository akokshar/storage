package handlers

import "net/http"

// Handler interface to be implemented by backend instances
type Handler interface {
	ServeHTTPRequest(w http.ResponseWriter, r *http.Request)
	GetRoutePrefix() string
	GetBaseDir() string
}
