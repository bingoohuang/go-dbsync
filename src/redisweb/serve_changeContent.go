package main

import (
	"net/http"
	"strings"
)

func serveChangeContent(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	key := strings.TrimSpace(req.FormValue("key"))
	changedContent := strings.TrimSpace(req.FormValue("changedContent"))
	format := strings.TrimSpace(req.FormValue("format"))

	ok := changeContent(key, changedContent, format)
	w.Write([]byte(ok))
}
