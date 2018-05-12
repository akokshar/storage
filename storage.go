package main

import (
	"os"
	"net/http"
	"fmt"
	"encoding/json"
	"net/url"
	"path"
	"strings"
	"io/ioutil"
)

const basedir = "/tmp"

type Response struct {
	Code int
	Body ResponseBody
}

type ResponseBody interface {
	toBytes() []byte
}

/*
 response body can be a Message  or a StorageContent
 Message and StorageContent do implement ResponseBody interface
 */

type Message struct {
	Msg string	`json:"msg"`
}

type StorageContent interface {
	 load()
}

/*
 StorageContent differs - it can be a file or a directory
 Both have some metadata (StorageFileInfo) and the actual content
 StorageFile implements ResponseBody interface
 */

type StorageFileInfo struct {
	Name		string		`json:"name"`
	IsDirectory	bool		`json:"directory"`
	ModDate		int64		`json:"create_date"`
	Size		int64		`json:"size"`
}

type StorageFile struct {
	StorageFileInfo
	Content		StorageContent	`json:"content"`
}

/*
 Actual content is represented by StorageFileContent and StorageDirContent
 these implemends StorageContent interface
 */

type StorageContentInfo struct {
	Offset	int		`json:"offset"`
	Count	int		`json:"count"`
}

type StorageFileContent struct {
	StorageContentInfo
	FileContent	string `json:"data"`
}

type StorageDirContent struct {
	StorageContentInfo
	Files	[]StorageFileInfo	`json:"files"`
}

// ResponseBody implementation

func (f *StorageFile) toBytes() []byte {
	bytes, _:= json.Marshal(f)
	return bytes
}

func (e *Message) toBytes() []byte {
	bytes, _ := json.Marshal(e)
	return bytes
}

// StorageContent implementation

func (d *StorageDirContent) load() {

}

func (f *StorageFileContent) load() {

}

func (r *Response) parseRequest(requestUri string) (filepath string, opts url.Values, err error) {
	if reqUrl, err := url.ParseRequestURI(requestUri); err == nil {
		filepath = path.Clean(basedir + reqUrl.Path)
		if strings.HasPrefix(filepath, basedir) {
			opts, _ := url.ParseQuery(reqUrl.RawQuery)
			return filepath, opts, nil
		}
	}
	r.Code = http.StatusBadRequest
	r.Body = &Message{Msg: err.Error()}
	return "", nil, err
}

func (r *Response) getContent(filepath string, opts url.Values) {
	fileInfo, err := os.Stat(filepath)

	if  err != nil {
		r.Code = http.StatusNotFound
		r.Body = &Message{Msg: err.Error()}
		return
	}

	var content StorageContent
	var size int64
	if fileInfo.IsDir() {
		content = &StorageDirContent{
			StorageContentInfo{
				Offset:0,
				Count:10,
				},
			nil,
		}
		files,_ := ioutil.ReadDir(filepath)
		size = int64(len(files))
	} else {
		content = &StorageFileContent{
			StorageContentInfo{
				Offset:0,
				Count:0,
			},
			"",
		}
		size = fileInfo.Size()
	}

	content.load()

	r.Code = http.StatusOK
	r.Body = &StorageFile{
		StorageFileInfo{
			Name:        fileInfo.Name(),
			IsDirectory: fileInfo.IsDir(),
			ModDate:     fileInfo.ModTime().Unix(),
			Size:        size,
		},
		content,
	}
}

func (r *Response) deleteContent(filepath string) {
	if err := os.Remove(filepath); err != nil {
		r.Code = http.StatusInternalServerError
		r.Body = &Message{Msg: err.Error()}
	} else {
		r.Code = http.StatusOK
		r.Body = &Message{filepath + " deleted"}
	}
}

func (r *Response) createContent(filepath string, content []byte) {
	if file, err := os.Create(filepath); err != nil {
		r.Code = http.StatusInternalServerError
		r.Body = &Message{Msg: err.Error()}
	} else {
		if content != nil {
			file.Write(content)
		}
		file.Close()
		r.Code = http.StatusOK
		r.Body = &Message{filepath + " created"}
	}
}

func main()  {
	http.HandleFunc("/", func (responseWriter http.ResponseWriter, request *http.Request) {
		response := &Response{Code:http.StatusMethodNotAllowed, Body:nil}

		if filepath, opts, err := response.parseRequest(request.RequestURI); err == nil {
			switch request.Method {
				case http.MethodGet: {
					response.getContent(filepath, opts)
				}
				case http.MethodDelete: {
					response.deleteContent(filepath)
				}
				case http.MethodPut: {
					response.createContent(filepath, nil)
				}
			}
		}

		responseWriter.WriteHeader(response.Code)
		if response.Body != nil {
			responseWriter.Write(response.Body.toBytes())
		}
	})

	fmt.Errorf("%v", http.ListenAndServe(":8080", nil))
}
