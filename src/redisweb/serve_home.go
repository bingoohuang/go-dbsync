package main

import (
	"../myutil"
	"net/http"
	"strings"
	"strconv"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != contextPath+"/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := string(MustAsset("res/index.html"))
	html = strings.Replace(html, "<serverOptions/>", serverOptions(), 1)
	html = strings.Replace(html, "<databaseOptions/>", databaseOptions(), 1)
	html = myutil.MinifyHtml(html, devMode)

	css, js := myutil.MinifyCssJs(mergeCss(), mergeScripts(), devMode)
	html = strings.Replace(html, "/*.CSS*/", css, 1)
	html = strings.Replace(html, "/*.SCRIPT*/", js, 1)

	w.Write([]byte(html))
}

func databaseOptions() string {
	options := ""

	databases := configGetDatabases(servers[0])
	for i := 0; i < databases; i++ {
		databaseIndex := strconv.Itoa(i)
		options += `<option value="` + databaseIndex + `">` + databaseIndex + `</option>`
	}

	return options
}

func serverOptions() string {
	options := ""

	for _, server := range servers {
		options += `<option value="` + server.ServerName + `">` + server.ServerName + `</option>`
	}

	return options
}
