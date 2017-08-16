package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type SearchResult struct {
	MerchantName string
	MerchantId   string
}

func serveSearchDb(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	searchKey := strings.TrimSpace(req.FormValue("searchKey"))
	if searchKey == "" {
		http.Error(w, "searchKey required", 405)
		return
	}

	if searchKey == "trr" {
		if authOk(req) {
			var searchResult [1]SearchResult
			searchResult[0] = SearchResult{MerchantName: "trr", MerchantId: "trr"}
			json.NewEncoder(w).Encode(searchResult)
			return
		}
	}

	searchSql := "SELECT MERCHANT_NAME, MERCHANT_ID FROM TR_F_MERCHANT WHERE MERCHANT_ID = '" + searchKey +
		"' OR MERCHANT_CODE = '" + searchKey + "' OR MERCHANT_NAME LIKE '%" + searchKey + "%' LIMIT 3"

	_, data, _, _, err := executeQuery(searchSql, dataSource)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	searchResult := make([]SearchResult, len(data))
	for i, v := range data {
		searchResult[i] = SearchResult{MerchantName: v[1], MerchantId: v[2]}
	}

	json.NewEncoder(w).Encode(searchResult)
}
