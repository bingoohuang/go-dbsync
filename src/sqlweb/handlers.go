package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xwb1989/sqlparser"
	"fmt"
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

	index := string(MustAsset("res/index.html"))
	index = strings.Replace(index, "/*.CSS*/", mergeCss(), 1)
	index = strings.Replace(index, "/*.SCRIPT*/", mergeScripts(), 1)

	w.Write([]byte(index))
}

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

	searchSql := `SELECT MERCHANT_NAME, MERCHANT_ID
		FROM TR_F_MERCHANT WHERE MERCHANT_ID = '` + searchKey + `'
		OR MERCHANT_CODE = '` + searchKey + `'
		OR MERCHANT_NAME LIKE '%` + searchKey + `%'
		LIMIT 3`

	_, data, _, _, err := executeQuery(searchSql, dataSource)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	searchResult := make([]SearchResult, len(data))
	for i, v := range data {
		searchResult[i] = SearchResult{v[1], v[2]}
	}

	json.NewEncoder(w).Encode(searchResult)
}

type QueryResult struct {
	Headers       []string
	Rows          [][]string
	Error         string
	ExecutionTime string
	CostTime      string
}

func serveQuery(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	querySql := strings.TrimSpace(req.FormValue("sql"))
	tid := strings.TrimSpace(req.FormValue("tid"))

	dbDataSource, err := selectDb(tid)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	start := time.Now()
	sqlParseResult, _ := sqlparser.Parse(querySql)

	switch sqlParseResult.(type) {
	case *sqlparser.Insert, *sqlparser.Delete, *sqlparser.Update, *sqlparser.Set:
		if writeAuthRequired {
			json.NewEncoder(w).Encode(QueryResult{Headers: nil, Rows: nil,
				Error: "dangerous sql, please get authorized first!",
				ExecutionTime: start.Format("2006-01-02 15:04:05.000"),
				CostTime: time.Since(start).String(),
			})
			log.Println("sql", querySql, "is not allowed because of insert/delete/update/set")
			return
		}
	case *sqlparser.Select:
		tableName := findSingleTableName(sqlParseResult)
		if tableName != "" {
			fmt.Println("table:", tableName)
			pris := make([]string, 0)
			_, data, _, _, err := executeQuery("desc " + tableName, dbDataSource)
			if err == nil {
				for i := 0; i < len(data); i++ {
					if data[i][4] == "PRI" {
						pris = append(pris, data[i][1])
					}
				}
 			}

			fmt.Println("pris:", pris)
		}
	}

	var (
		header        []string
		data          [][]string
		executionTime string
		costTime      string
	)

	isShowHistory := strings.EqualFold("show history", querySql)
	if isShowHistory {
		header, data, executionTime, costTime, err = showHistory()
	} else {
		saveHistory(querySql)
		header, data, executionTime, costTime, err = executeQuery(querySql, dbDataSource)
	}
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	queryResult := QueryResult{Headers: header, Rows: data, Error: errMsg, ExecutionTime: executionTime, CostTime: costTime}

	json.NewEncoder(w).Encode(queryResult)
}

func findSingleTableName(sqlParseResult sqlparser.Statement) string {
	selectSql, _ := sqlParseResult.(*sqlparser.Select)
	if len(selectSql.From) != 1 {
		return ""
	}

	tableExpr := selectSql.From[0]
	aliasTableExpr, ok := tableExpr.(*sqlparser.AliasedTableExpr)
	if !ok {
		return ""
	}

	simpleTableExpr, ok := aliasTableExpr.Expr.(*sqlparser.TableName)
	if !ok {
		return ""
	}

	return string(simpleTableExpr.Name)
}
