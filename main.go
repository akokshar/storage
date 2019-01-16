package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/akokshar/storage/server"
)

const (
	paramBasedirName = "basedir"
	defaultBasedir   = "/tmp/Store"
	paramPortName    = "port"
	defaultPort      = "8083"
)

func main() {
	var basedir string
	var port string

	flag.StringVar(&basedir, paramBasedirName, "", "The directory with files")
	flag.StringVar(&port, paramPortName, "", "Listen port")
	flag.Parse()

	if basedir == "" {
		basedir, _ = os.LookupEnv(strings.ToUpper(paramBasedirName))
		if basedir == "" {
			basedir = defaultBasedir
		}
	}

	if port == "" {
		port, _ = os.LookupEnv(strings.ToUpper(paramPortName))
		if port == "" {
			port = defaultPort
		}
	}

	log.Printf("Storage is about to serve `%s` on port `%s`\n", basedir, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), server.CreateApplication(basedir)))
}
