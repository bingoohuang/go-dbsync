package main

import (
	"net/http"
	"strings"
)

func serveDeleteKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	key := strings.TrimSpace(req.FormValue("key"))
	server := findRedisServer(req)

	ok := deleteKey(server, key)
	w.Write([]byte(ok))
}
