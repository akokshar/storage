package meta

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/akokshar/storage/server/handlers"
)

const (
	enumerateOptName     = "enumerate"
	countOptName         = "count"
	countDefaultValue    = 10
	offsetOptName        = "offset"
	offsetDefaultValue   = 0
	contentTypeDirectory = "folder"
)

type meta struct {
	routePrefix string
	basedir     string
}

type fileMeta struct {
	Name  string `json:"nn"`
	CType string `json:"ct"`
	Size  int64  `json:"sz"`
	MDate int64  `json:"md"`
	CDate int64  `json:"cd"`
}

type dirMeta struct {
	Offset int         `json:"offset"`
	Files  []*fileMeta `json:"files"`
}

func getDirContent(dir string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Printf("Failed to read directory '%s' due to '%s'", dir, err.Error())
		return nil, err
	}
	filtered := files[:0]
	for _, f := range files {
		if f.Mode().IsDir() || f.Mode().IsRegular() {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

func newFileMeta(fPath string) (*fileMeta, handlers.HandlerError) {
	fi, err := os.Stat(fPath)
	if err != nil {
		log.Printf("Stat '%s' failed with '%s'", fPath, err.Error())
		return nil, handlers.NewHandlerErrorWithCode(http.StatusNotFound)
	}

	fMeta := new(fileMeta)
	fMeta.Name = fi.Name()
	if fi.IsDir() {
		fMeta.CType = contentTypeDirectory
		dirContent, err := getDirContent(fPath)
		if err != nil {
			log.Printf("Failed to read directory '%s' due to '%s'", fPath, err.Error())
			return nil, handlers.NewHandlerErrorWithCode(http.StatusInternalServerError)
		}
		fMeta.Size = int64(len(dirContent))
	} else {
		fMeta.CType = "application/octet-stream"
		f, err := os.Open(fPath)
		if err != nil {
			log.Printf("Failed to open '%s' to determine its content type due to '%s'", fPath, err.Error())
			return nil, handlers.NewHandlerErrorWithCode(http.StatusInternalServerError)
		}
		defer f.Close()
		buffer := make([]byte, 512)
		if count, _ := f.Read(buffer); count < 512 {
			if cType := mime.TypeByExtension(filepath.Ext(fi.Name())); cType != "" {
				fMeta.CType = cType
			}
		} else {
			fMeta.CType = http.DetectContentType(buffer)
		}
		fMeta.Size = fi.Size()
	}

	fMeta.CDate, fMeta.MDate = getFileTimeStamps(fi)

	return fMeta, nil
}

func newDirMeta(dPath string, offset int, count int) (*dirMeta, handlers.HandlerError) {
	fi, err := os.Stat(dPath)
	if err != nil {
		log.Printf("Stat '%s' failed with '%s'", dPath, err.Error())
		return nil, handlers.NewHandlerErrorWithCode(http.StatusNotFound)
	}

	if !fi.IsDir() {
		return nil, handlers.NewHandlerErrorWithCode(http.StatusNoContent)
	}

	dirContent, err := getDirContent(dPath)
	if err != nil {
		return nil, handlers.NewHandlerErrorWithCode(http.StatusInternalServerError)
	}

	if offset > len(dirContent) {
		log.Printf("Out of range enumeration of '%s'", dPath)
		return nil, handlers.NewHandlerErrorWithCode(http.StatusNoContent)
	}

	if tailLen := len(dirContent) - offset; tailLen < count {
		count = tailLen
	}

	sort.SliceStable(dirContent, func(i int, j int) bool {
		if dirContent[i].IsDir() == dirContent[j].IsDir() {
			return strings.ToLower(dirContent[i].Name()) < strings.ToLower(dirContent[j].Name())
		}
		return dirContent[i].IsDir()
	})

	dMeta := new(dirMeta)
	dMeta.Offset = offset
	dMeta.Files = make([]*fileMeta, 0, count)
	for i := 0; i < count; i++ {
		fPath := path.Join(dPath, dirContent[i].Name())
		fMeta, err := newFileMeta(fPath)
		if err == nil {
			dMeta.Files = append(dMeta.Files, fMeta)
		}
	}
	return dMeta, nil
}

// New initializes backend to serve metadata
func New(prefix string, basedir string) handlers.Handler {
	return &meta{
		routePrefix: prefix,
		basedir:     basedir,
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

	localFilepath := r.Header.Get("X-Local-Filepath")

	var metaData interface{}
	var e handlers.HandlerError

	if opts.Get(enumerateOptName) == "" {
		metaData, e = newFileMeta(localFilepath)
	} else {
		var offset, count int
		if offset, err = strconv.Atoi(opts.Get(offsetOptName)); err != nil {
			offset = offsetDefaultValue
		}
		if count, err = strconv.Atoi(opts.Get(countOptName)); err != nil {
			count = countDefaultValue
		}
		metaData, e = newDirMeta(localFilepath, offset, count)
	}

	if e != nil {
		w.WriteHeader(e.Code())
		return
	}

	metaJSON, _ := json.Marshal(metaData) // Indent(metaData, "", "  ")
	w.Header().Add("Content-Type", "application/json")
	w.Write(metaJSON)
}
