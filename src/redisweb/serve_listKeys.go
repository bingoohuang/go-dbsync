package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"strconv"
)

func serveListKeys(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	server := findRedisServer(req)

	keys, _ := listKeys(server, "", 100)
	sort.Slice(keys[:], func(i, j int) bool {
		return keys[i].Key < keys[j].Key
	})
	json.NewEncoder(w).Encode(keys)
}

func findRedisServer(req *http.Request) RedisServer {
	serverName := strings.TrimSpace(req.FormValue("serverName"))
	database := strings.TrimSpace(req.FormValue("database"))
	server := findServer(serverName)
	server.DB, _ = strconv.Atoi(database)
	return server
}

func findServer(serverName string) RedisServer {
	for _, server := range servers {
		if server.ServerName == serverName {
			return server
		}
	}

	return servers[0]
}
