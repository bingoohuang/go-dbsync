package main

import (
	"log"
	"net/http"
	"strings"
)

func serveNewKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	keyType := strings.TrimSpace(req.FormValue("type"))
	key := strings.TrimSpace(req.FormValue("key"))
	ttl := strings.TrimSpace(req.FormValue("ttl"))
	format := strings.TrimSpace(req.FormValue("format"))
	value := strings.TrimSpace(req.FormValue("value"))

	server := findRedisServer(req)

	log.Println("keyType:", keyType, ",key:", key, ",ttl:", ttl, ",format:", format, ",value:", value)

	ok := newKey(server, keyType, key, ttl, value)
	w.Write([]byte(ok))
}
