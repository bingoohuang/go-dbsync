package main

import (
	"net/http"
	"encoding/json"
	"strings"
)

type ContentResult struct {
	Content string
	Ttl     string
}

func serveShowContent(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	key := strings.TrimSpace(req.FormValue("key"))
	valType := strings.TrimSpace(req.FormValue("type"))

	content, _ := displayContent(key, valType)
	ttl, _ := ttlContent(key)

	json.NewEncoder(w).Encode(ContentResult{
		Content: content,
		Ttl:     ttl.String(),
	})
}
