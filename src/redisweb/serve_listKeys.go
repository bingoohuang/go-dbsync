package main

import (
	"encoding/json"
	"net/http"
)

func serveListKeys(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	keys, _ := listKeys("", 100)
	json.NewEncoder(w).Encode(keys)
}
