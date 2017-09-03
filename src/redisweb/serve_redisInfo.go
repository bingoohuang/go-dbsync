package main

import (
	"net/http"
	"strings"
)

func serveRedisCli(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	server := findRedisServer(req)
	cmd := strings.TrimSpace(req.FormValue("cmd"))

	result := repl(server, cmd)
	w.Write([]byte(result))
}
func serveRedisInfo(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	server := findRedisServer(req)

	//log.Println("keyType:", keyType, ",key:", key, ",ttl:", ttl, ",format:", format, ",value:", value)

	ok := redisInfo(server)
	w.Write([]byte(ok))
}
