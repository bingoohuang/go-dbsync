package main

import (
	"net/http"
	"strings"
	"strconv"
	"encoding/json"
)

type UpdateResultRow struct {
	Ok      bool
	Message string
}

type UpdateResult struct {
	Ok         bool
	Message    string
	RowsResult []UpdateResultRow
}

func serveUpdate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if writeAuthRequired {
		updateResult := UpdateResult{Ok: false, Message: "auth required!"}
		json.NewEncoder(w).Encode(updateResult)
		return
	}
	sqls := strings.TrimSpace(req.FormValue("sqls"))
	tid := strings.TrimSpace(req.FormValue("tid"))

	dbDataSource, err := selectDb(tid)
	if err != nil {
		updateResult := UpdateResult{Ok: false, Message: err.Error()}
		json.NewEncoder(w).Encode(updateResult)
		return
	}

	resultRows := make([]UpdateResultRow, 0)
	for _, sql := range strings.Split(sqls, ";\n") {
		_, _, rowsAffected, err := executeUpdate(sql, dbDataSource)
		if err != nil {
			resultRows = append(resultRows, UpdateResultRow{Ok: false, Message: err.Error()})
		} else if rowsAffected == 1 {
			resultRows = append(resultRows, UpdateResultRow{Ok: true, Message: "1 rows affected!"})
		} else {
			resultRows = append(resultRows, UpdateResultRow{Ok: false, Message: strconv.FormatInt(rowsAffected, 10) + " rows affected!"})
		}
	}

	updateResult := UpdateResult{Ok: true, Message: "Ok", RowsResult: resultRows}
	json.NewEncoder(w).Encode(updateResult)
}
