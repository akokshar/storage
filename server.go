package main

import (
	"flag"
	"net/http"
	"log"
	"net/url"
	"encoding/json"
)

const defaultBasedir = "/tmp"

type response struct {
	code int
	body []byte
}

func (r *response) serErrorMessage(code int, message string) {
	jsonMsg, _ := json.Marshal(struct {Msg string}{Msg:message})
	r.code = code
	r.body = jsonMsg
}

func (r *response) setBody() {
	r.code = http.StatusOK
}

func main() {
	var basedir	string
	flag.StringVar(&basedir, "basedir", defaultBasedir, "The directory with files")
	flag.Parse()

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		var r response

		reqURL, err := url.ParseRequestURI(request.RequestURI)
		if err != nil {
			r.serErrorMessage(http.StatusBadRequest, err.Error())
		} else {
			storeItem, err := InitFileStoreItem(basedir, reqURL.Path)
			if err != nil {
				r.serErrorMessage(http.StatusForbidden, err.Error())
			}

			switch request.Method {
			case http.MethodGet:
				{
					opts := make(map[string]string)
					if reqOpts, err := url.ParseQuery(reqURL.RawQuery); err == nil {
						if offset := reqOpts[DirListOffsetOptName]; offset != nil {
							opts[DirListOffsetOptName] = offset[0]
						}
						if count := reqOpts[DirListCountOptName]; count != nil {
							opts[DirListCountOptName] = count[0]
						}
					}
					r.body, err = storeItem.GetJson(opts)
					if err != nil {
						r.serErrorMessage(http.StatusInternalServerError, err.Error())
					} else {
						r.code = http.StatusOK
					}
				}
			case http.MethodDelete:
				{
					if err := storeItem.Delete(); err != nil {
						r.serErrorMessage(http.StatusInternalServerError, err.Error())
					} else {
						r.serErrorMessage(http.StatusOK, "deleted")
					}
				}
			case http.MethodPost:
				{
					if err := storeItem.CreateFile(nil); err != nil {
						r.serErrorMessage(http.StatusInternalServerError, err.Error())
					} else {
						r.serErrorMessage(http.StatusOK, "created")
					}
				}
			default:
				{
					r.serErrorMessage(http.StatusMethodNotAllowed, "Metod not allowed")
				}

			}

		}
		responseWriter.WriteHeader(r.code)
		responseWriter.Write(r.body)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
