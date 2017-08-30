package main

import (
	"../myutil"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

var (
	contextPath string
	port        int

	devMode bool // to disable css/js minify
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 8269, "Port to serve.")
	devModeArg := flag.Bool("devMode", false, "devMode(disable js/css minify)")

	flag.Parse()

	contextPath = *contextPathArg
	port = *portArg
	devMode = *devModeArg
}

func main() {
	http.HandleFunc(contextPath+"/", myutil.GzipWrapper(serveHome))
	http.HandleFunc(contextPath+"/favicon.png", serveImage("favicon.png"))
	http.HandleFunc(contextPath+"/spritesheet.png", serveImage("spritesheet.png"))
	http.HandleFunc(contextPath+"/listKeys", serveListKeys)
	http.HandleFunc(contextPath+"/showContent", serveShowContent)
	http.HandleFunc(contextPath+"/changeContent", serveChangeContent)
	http.HandleFunc(contextPath+"/deleteKey", serveDeleteKey)

	sport := strconv.Itoa(port)
	fmt.Println("start to listen at ", sport)
	if err := http.ListenAndServe(":"+sport, nil); err != nil {
		log.Fatal(err)
	}
}
