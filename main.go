package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/akokshar/storage/server"
)

const (
	defaultBasedir = "/tmp"
	defaultPort    = "8082"
)

func main() {
	var basedir string
	var port string

	flag.StringVar(&basedir, "basedir", defaultBasedir, "The directory with files")
	flag.StringVar(&port, "port", defaultPort, "Listen port")
	flag.Parse()

	log.Printf("Storage is about to serve `%s` on port `%s`\n", basedir, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), server.CreateApplication(basedir)))
}
