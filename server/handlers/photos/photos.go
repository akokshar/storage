package photos

import (
	"net/http"

	"github.com/akokshar/storage/server/handlers"
)

type photos struct {
	routePrefix string
	basedir     string
}

// New initializes backend to server files
func New(prefix string, basedir string) handlers.Handler {
	p := &photos{
		routePrefix: prefix,
		basedir:     basedir,
	}

	return p
}

func (p *photos) GetRoutePrefix() string {
	return p.routePrefix
}

func (p *photos) GetBaseDir() string {
	return p.basedir
}

func (p *photos) ServeHTTPRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
