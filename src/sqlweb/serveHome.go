package main

import (
	"net/http"
	"strings"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != contextPath+"/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	index := string(MustAsset("res/index.html"))
	index = strings.Replace(index, "/*.CSS*/", mergeCss(), 1)
	index = strings.Replace(index, "/*.SCRIPT*/", mergeScripts(), 1)

	w.Write([]byte(index))
}
