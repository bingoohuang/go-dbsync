package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func serveShowContent(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	key := strings.TrimSpace(req.FormValue("key"))
	valType := strings.TrimSpace(req.FormValue("type"))

	content, _ := displayContent(key, valType)
	json.NewEncoder(w).Encode(content)
}
