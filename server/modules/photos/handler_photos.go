package photos

import (
	"net/http"

	"github.com/akokshar/storage/server/modules"
)

type photos struct {
	routePrefix string
	basedir     string
}

// New initializes backend to server files
func New(prefix string, basedir string) modules.HTTPHandler {
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
