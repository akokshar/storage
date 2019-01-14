package meta

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/akokshar/storage/server/backend"
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
	prefix string
}

type fileMeta struct {
	Name  string `json:"name"`
	CType string `json:"c_type"`
	Size  int64  `json:"size"`
	MDate int64  `json:"m_date"`
	CDate int64  `json:"c_date"`
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

func newFileMeta(fPath string) (*fileMeta, backend.HandlerError) {
	fi, err := os.Stat(fPath)
	if err != nil {
		log.Printf("Stat '%s' failed with '%s'", fPath, err.Error())
		return nil, backend.NewHandlerErrorWithCode(http.StatusNotFound)
	}

	fMeta := new(fileMeta)
	fMeta.Name = fi.Name()
	if fi.IsDir() {
		fMeta.CType = contentTypeDirectory
		dirContent, err := getDirContent(fPath)
		if err != nil {
			log.Printf("Failed to read directory '%s' due to '%s'", fPath, err.Error())
			return nil, backend.NewHandlerErrorWithCode(http.StatusInternalServerError)
		}
		fMeta.Size = int64(len(dirContent))
	} else {
		f, err := os.Open(fPath)
		if err != nil {
			log.Printf("Failed to open '%s' to determine its content type due to '%s'", fPath, err.Error())
			return nil, backend.NewHandlerErrorWithCode(http.StatusInternalServerError)
		}
		defer f.Close()
		buffer := make([]byte, 512)
		_, err = f.Read(buffer)
		if err != nil {
			log.Printf("Read from '%s' failed due to '%s'", fPath, err.Error())
			return nil, backend.NewHandlerErrorWithCode(http.StatusInternalServerError)
		}
		fMeta.Size = fi.Size()
	}
	sysStat := fi.Sys().(*syscall.Stat_t)
	//fMeta.MDate = sysStat.Mtim.Sec
	//fMeta.CDate = sysStat.Ctim.Sec
	fMeta.MDate = sysStat.Mtimespec.Sec
	fMeta.CDate = sysStat.Ctimespec.Sec

	return fMeta, nil
}

func newDirMeta(dPath string, offset int, count int) (*dirMeta, backend.HandlerError) {
	fi, err := os.Stat(dPath)
	if err != nil {
		log.Printf("Stat '%s' failed with '%s'", dPath, err.Error())
		return nil, backend.NewHandlerErrorWithCode(http.StatusNotFound)
	}

	if !fi.IsDir() {
		return nil, backend.NewHandlerErrorWithCode(http.StatusNoContent)
	}

	dirContent, err := getDirContent(dPath)
	if err != nil {
		return nil, backend.NewHandlerErrorWithCode(http.StatusInternalServerError)
	}

	if offset > len(dirContent) {
		log.Printf("Out of range enumeration of '%s'", dPath)
		return nil, backend.NewHandlerErrorWithCode(http.StatusNoContent)
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
	dMeta.Files = make([]*fileMeta, count)
	for i := 0; i < count; i++ {
		fPath := path.Join(dPath, dirContent[i].Name())
		dMeta.Files[i], _ = newFileMeta(fPath)
	}
	return dMeta, nil
}

// New initializes backend to serve metadata
func New() backend.Handler {
	return &meta{
		prefix: "/meta/",
	}
}

func (m *meta) GetRoutePrefix() string {
	return m.prefix
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
	var e backend.HandlerError

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

	metaJSON, _ := json.MarshalIndent(metaData, "", "  ")
	w.Header().Add("Content-Type", "application/json")
	w.Write(metaJSON)
}
