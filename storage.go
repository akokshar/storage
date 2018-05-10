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

type ResponseBody interface {
	toBytes() []byte
}

type Response struct {
	Code int
	Body ResponseBody
}

type DataBody struct {
	Name string			`json:"name"`
	IsDirectory bool	`json:"directory"`
	ModDate int64		`json:"create_date"`
	Size int64			`json:"size"`
}

type ErrorBody struct {
	Msg string	`json:msg`
}

func (d *DataBody) toBytes() []byte {
	bytes, _:= json.Marshal(d)
	return bytes
}

func (e *ErrorBody) toBytes() []byte {
	bytes, _ := json.Marshal(e)
	return bytes
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
	r.Body = &ErrorBody{Msg: err.Error()}
	return "", nil, err
}

func (r *Response) getContent(filepath string, opts url.Values) {
	fileInfo, err := os.Stat(filepath)

	if  err != nil {
		r.Code = http.StatusNotFound
		r.Body = &ErrorBody{Msg: err.Error()}
		return
	}

	var size int64
	if fileInfo.IsDir() {
		files,_ := ioutil.ReadDir(filepath)
		size = int64(len(files))
	} else {
		size = fileInfo.Size()
	}

	r.Code = http.StatusOK
	r.Body = &DataBody{
		Name:fileInfo.Name(),
		IsDirectory:fileInfo.IsDir(),
		ModDate:fileInfo.ModTime().Unix(),
		Size:size,
	}
}

func (r *Response) deleteContent(filepath string) {
	if err := os.Remove(filepath); err != nil {
		r.Code = http.StatusInternalServerError
		r.Body = &ErrorBody{Msg: err.Error()}
	} else {
		r.Code = http.StatusOK
		r.Body = &ErrorBody{filepath + " deleted"}
	}
}

func (r *Response) createContent(filepath string, content []byte) {
	if file, err := os.Create(filepath); err != nil {
		r.Code = http.StatusInternalServerError
		r.Body = &ErrorBody{Msg: err.Error()}
	} else {
		if content != nil {
			file.Write(content)
		}
		file.Close()
		r.Code = http.StatusOK
		r.Body = &ErrorBody{filepath + " created"}
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
