package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
)

const defaultBasedir = "/tmp"
const defaultPort = "8080"

const dirListOffsetOptName = "offset"
const defaultDirListOffset = 0

const dirListCountOptName = "count"
const defaultDirListCount = 10

const dirCreateOptName = "directory"
const getItemInfoOptName = "info"

var basedir string
var port string

type storeItemInfo struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"is_dir"`
	ModDate     int64  `json:"m_date"`
	Size        int64  `json:"size"`

    itemPath    string
    files       []os.FileInfo
}

type storeDirContent struct {
	Offset int              `json:"offset"`
	Files  []*storeItemInfo `json:"files"`
}

type server struct {
	responseWriter http.ResponseWriter
	request        *http.Request
	reqURL         *url.URL
	itemFilePath   string
}

func getSortedDirContent(dirPath string) ([]os.FileInfo, error) {
	dirContent, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(dirContent, func(i int, j int) bool {
		if dirContent[i].IsDir() == dirContent[j].IsDir() {
			return strings.ToLower(dirContent[i].Name()) < strings.ToLower(dirContent[j].Name())
		}
		return dirContent[i].IsDir()
	})
	return dirContent, nil
}

func (itemInfo *storeItemInfo) initWithPath(itemPath string) error {
	fi, err := os.Stat(itemPath)
	if err != nil {
		return err
	}

    itemInfo.Name = fi.Name()
	itemInfo.IsDirectory = fi.IsDir()
	itemInfo.ModDate = fi.ModTime().Unix()
    itemInfo.itemPath = itemPath

	var size int64
	if fi.IsDir() {
        files, _ := getSortedDirContent(itemInfo.itemPath)
		itemInfo.files = files
		size = int64(len(itemInfo.files))
	} else {
        itemInfo.files = nil
		size = fi.Size()
	}
	itemInfo.Size = size

    return nil
}

func (itemInfo *storeItemInfo) getItemAtIndex(index int) *storeItemInfo {
    if index >= len(itemInfo.files) || index < 0 {
        return nil
    }

    var childItemInfo storeItemInfo
    childItemInfo.initWithPath(path.Join(itemInfo.itemPath, itemInfo.files[index].Name()))
    return &childItemInfo
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
    opts, err := url.ParseQuery(s.reqURL.RawQuery)
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

    var itemInfo storeItemInfo
    if err := itemInfo.initWithPath(s.itemFilePath); err != nil {
		s.responseWriter.WriteHeader(http.StatusNotFound)
		return
    }

    if opts[getItemInfoOptName] != nil {
		itemInfoJSON, _ := json.MarshalIndent(itemInfo, "", "  ")
		s.responseWriter.Header().Add("Content-Type", "application/json")
		s.responseWriter.Write(itemInfoJSON)
        return
    }

	if !itemInfo.IsDirectory {
		http.ServeFile(s.responseWriter, s.request, itemInfo.itemPath)
        return
    }

	var offset int
    if reqOffset := opts[dirListOffsetOptName]; reqOffset != nil {
        if offset, err = strconv.Atoi(reqOffset[0]); err != nil {
            offset = defaultDirListOffset
        }
    } else {
        offset = defaultDirListOffset
    }

    var count int
    if reqCount := opts[dirListCountOptName]; reqCount != nil {
        if count, err = strconv.Atoi(reqCount[0]); err != nil {
            count = defaultDirListCount
        }
    } else {
        count = defaultDirListCount
	}
	if tailLen := len(itemInfo.files) - offset; tailLen < count {
		count = tailLen
	}

	dirFilesInfo := make([]*storeItemInfo, count)
	for i := 0; i < count; i++ {
        dirFilesInfo[i] = itemInfo.getItemAtIndex(i + offset)
    }

    content := &storeDirContent{
		Offset: offset,
		Files:  dirFilesInfo,
	}

	contentJSON, _ := json.MarshalIndent(content, "", "  ")
	s.responseWriter.Header().Add("Content-Type", "application/json")
	s.responseWriter.Write(contentJSON)
}

func (s *server) createDir() {
	err := os.Mkdir(s.itemFilePath, 0755)
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	pPath, dName := path.Split(s.itemFilePath)
	pFiles, err := getSortedDirContent(pPath)
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	sFunc := func(i int) bool {
		if pFiles[i].IsDir() {
			return strings.ToLower(pFiles[i].Name()) >= strings.ToLower(dName)
		}
		return true
	}
	pos := sort.Search(len(pFiles), sFunc)
	content := &storeDirContent{
		Offset: pos,
		Files: []*storeItemInfo{
			&storeItemInfo{
				Name:        dName,
				IsDirectory: true,
				ModDate:     pFiles[pos].ModTime().Unix(),
				Size:        0,
			},
		},
	}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.responseWriter.Header().Add("Content-Type", "application/json")
	s.responseWriter.Write(contentJSON)
	return
}

func (s *server) createFile() {
	f, err := os.Create(s.itemFilePath)
	if err != nil {
		e, _ := err.(*os.PathError)
		switch e.Err {
		case syscall.EACCES:
			{
				s.responseWriter.WriteHeader(http.StatusForbidden)
			}
		case syscall.ENOENT:
			{
				s.responseWriter.WriteHeader(http.StatusNoContent)
			}
		default:
			{
				s.responseWriter.WriteHeader(http.StatusInternalServerError)
			}
		}
		return
	}
	defer f.Close()

	_, err = io.Copy(f, s.request.Body)
// TODO:
// ContentLen condition does not work for binary data. to investigate later. 
//	if err != nil || n != s.request.ContentLength {
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

    var fileInfo storeItemInfo
    fileInfo.initWithPath(s.itemFilePath)

	fileInfoJSON, err := json.Marshal(fileInfo)
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.responseWriter.Header().Add("Content-Type", "application/json")
	s.responseWriter.WriteHeader(http.StatusCreated)
	s.responseWriter.Write(fileInfoJSON)
}

func (s *server) processPost() {
	opts, err := url.ParseQuery(s.reqURL.RawQuery)
	if err != nil {
		s.responseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(s.itemFilePath); err == nil {
		s.responseWriter.WriteHeader(http.StatusConflict)
		return
	}

	createDir := opts[dirCreateOptName] != nil
	if createDir {
		s.createDir()
	} else {
		s.createFile()
	}
}

func (s *server) processDelete() {
	if err := os.Remove(s.itemFilePath); err != nil {
		log.Print(err.Error())
		s.responseWriter.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) processRequest() {
	switch s.request.Method {
	case http.MethodGet:
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
	flag.StringVar(&port, "port", defaultPort, "Listen port")
	flag.Parse()

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		var s server
		if err := s.initWith(responseWriter, request); err == nil {
			s.processRequest()
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
