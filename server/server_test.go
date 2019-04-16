package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akokshar/storage/server/modules"
)

type module struct {
	prefix  string
	baseDir string
}

func (m *module) GetRoutePrefix() string {
	return m.prefix
}

func (m *module) GetBaseDir() string {
	return m.baseDir
}

func (m *module) ServeHTTPRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func createTestApplication() application {
	app := application{
		handlers: make([]modules.HTTPHandler, 0, 1),
		filesDB:  nil,
	}

	app.handlers = append(
		app.handlers,
		&module{
			prefix:  "/test",
			baseDir: "/tmp/test/dir",
		},
	)
	return app
}

func TestServerModulePath(t *testing.T) {
	app := createTestApplication()
	request, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ServeHTTP)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Error("/test route failed")
	}
}

func TestServerEscapeModulePath(t *testing.T) {
	app := createTestApplication()
	request, err := http.NewRequest("GET", "../test", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ServeHTTP)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Error("../test route failed")
	}
}

func TestServerEscapeModulePath2(t *testing.T) {
	app := createTestApplication()
	request, err := http.NewRequest("GET", "/test/../../", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ServeHTTP)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Error("/test/../../ route failed")
	}
}

func TestServerWrongModulePath(t *testing.T) {
	app := createTestApplication()
	request, err := http.NewRequest("GET", "/wrongpath", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(app.ServeHTTP)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Error("/wrongpath route failed")
	}
}
