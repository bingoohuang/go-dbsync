package main

import (
	"encoding/json"
	"net/http"
	"sort"
)

func serveListKeys(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	keys, _ := listKeys("", 100)
	sort.Slice(keys[:], func(i, j int) bool {
		return keys[i].Key < keys[j].Key
	})
	json.NewEncoder(w).Encode(keys)
}
