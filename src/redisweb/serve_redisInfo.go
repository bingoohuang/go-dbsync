package main

import (
	"net/http"
)

func serveRedisInfo(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	server := findRedisServer(req)

	//log.Println("keyType:", keyType, ",key:", key, ",ttl:", ttl, ",format:", format, ",value:", value)

	ok := redisInfo(server)
	w.Write([]byte(ok))
}
