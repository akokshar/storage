package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

const defaultBasedir = "/tmp"

var basedir string

const DirListOffsetOptName = "offset"
const defaultDirListOffset = 0

const DirListCountOptName = "count"
const defaultDirListCount = 10

type storeItemInfo struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"directory"`
	ModDate     int64  `json:"create_date"`
	Size        int64  `json:"size"`

	files []os.FileInfo
}

type storeDirContent struct {
	Offset int              `json:"offset"`
	Count  int              `json:"count"`
	Files  []*storeItemInfo `json:"files"`
}

type storeDirInfo struct {
	storeItemInfo
	Content storeDirContent `json:"content"`
}

type server struct {
	responseWriter http.ResponseWriter
	request        *http.Request
	reqURL         *url.URL
	itemFilePath   string
}

func (s *server) initWith(responseWriter http.ResponseWriter, request *http.Request) error {
	s.responseWriter = responseWriter
	s.request = request

	var err error
	if s.reqURL, err = url.ParseRequestURI(s.request.RequestURI); err != nil {
		log.Print(err.Error())
		s.responseWriter.WriteHeader(http.StatusBadRequest)
		return err
	}

	s.itemFilePath = path.Join(basedir, s.reqURL.Path)
	if !strings.HasPrefix(s.itemFilePath, basedir) {
		err := errors.New("Wrong path")
		log.Print(err.Error())
		s.responseWriter.WriteHeader(http.StatusForbidden)
		return err
	}

	return nil
}

func (s *server) processGet() {
	itemFileInfo, err := os.Stat(s.itemFilePath)

	if err != nil {
		s.responseWriter.WriteHeader(http.StatusNotFound)
		return
	}

	if itemFileInfo.IsDir() {
		dirFiles, _ := ioutil.ReadDir(s.itemFilePath)
		dirSize := len(dirFiles)

		var offset, count int
		if reqOpts, err := url.ParseQuery(s.reqURL.RawQuery); err == nil {
			if reqOffset := reqOpts[DirListOffsetOptName]; reqOffset != nil {
				if offset, err = strconv.Atoi(reqOffset[0]); err != nil {
					offset = defaultDirListOffset
				}
			} else {
				offset = defaultDirListOffset
			}
			if reqCount := reqOpts[DirListCountOptName]; reqCount != nil {
				if count, err = strconv.Atoi(reqCount[0]); err != nil {
					count = defaultDirListCount
				}
			} else {
				count = defaultDirListCount
			}
		}
		if tailLen := dirSize - offset; tailLen < count {
			count = tailLen
		}

		dirFilesInfo := make([]*storeItemInfo, count)
		for i := 0; i < count; i++ {
			childFilePath := path.Join(s.itemFilePath, dirFiles[i+offset].Name())
			if childFileInfo, err := os.Stat(childFilePath); err == nil {
				var childSize int64
				if childFileInfo.IsDir() {
					childContent, _ := ioutil.ReadDir(childFilePath)
					childSize = int64(len(childContent))
				} else {
					childSize = childFileInfo.Size()
				}

				dirFilesInfo[i] = &storeItemInfo{
					Name:        childFileInfo.Name(),
					IsDirectory: childFileInfo.IsDir(),
					ModDate:     childFileInfo.ModTime().Unix(),
					Size:        childSize,
				}
			}
		}

		dir := &storeDirInfo{
			storeItemInfo{
				Name:        itemFileInfo.Name(),
				IsDirectory: true,
				ModDate:     itemFileInfo.ModTime().Unix(),
				Size:        int64(dirSize),
			},
			storeDirContent{
				Offset: offset,
				Count:  count,
				Files:  dirFilesInfo,
			},
		}

		dirJson, _ := json.MarshalIndent(dir, "", "  ")
		s.responseWriter.Header().Add("Content-Type", "application/json")
		s.responseWriter.Write(dirJson)

	} else {
		if file, err := os.Open(s.itemFilePath); err == nil {
			buf := make([]byte, itemFileInfo.Size())
			if _, err := file.Read(buf); err != nil {
				log.Print(err.Error())
				s.responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			s.responseWriter.Header().Add("Content-Type", http.DetectContentType(buf))
			s.responseWriter.Write(buf)
		}
	}
}

func (s *server) processPost() {

}

func (s *server) processDelete() {
	if err := os.Remove(s.itemFilePath); err != nil {
		log.Print(err.Error())
		s.responseWriter.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) processReqiest() {
	switch s.request.Method {
	case s.request.Method:
		{
			s.processGet()
		}
	case http.MethodPost:
		{
			s.processPost()
		}
	case http.MethodDelete:
		{
			s.processDelete()
		}
	default:
		{
			s.responseWriter.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func main() {
	flag.StringVar(&basedir, "basedir", defaultBasedir, "The directory with files")
	flag.Parse()

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		var s server
		if err := s.initWith(responseWriter, request); err == nil {
			s.processReqiest()
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
