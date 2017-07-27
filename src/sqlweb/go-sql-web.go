package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
)

var (
	contextPath       string
	port              int
	maxRows           int
	dataSource        string
	writeAuthRequired bool
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 8381, "Port to serve.")
	maxRowsArg := flag.Int("maxRows", 1000, "Max number of rows to return.")
	dataSourceArg := flag.String("dataSource", "user:pass@tcp(127.0.0.1:3306)/db?charset=utf8", "dataSource string.")
	writeAuthRequiredArg := flag.Bool("writeAuthRequired", true, "write auth required")

	flag.Parse()

	contextPath = *contextPathArg
	port = *portArg
	maxRows = *maxRowsArg
	dataSource = *dataSourceArg
	writeAuthRequired = *writeAuthRequiredArg
}

func main() {
	http.HandleFunc(contextPath+"/", gzipWrapper(serveHome))
	http.HandleFunc(contextPath+"/query", serveQuery)
	http.HandleFunc(contextPath+"/searchDb", serveSearchDb)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}
