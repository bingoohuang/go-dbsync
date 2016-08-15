package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"reflect"
	"strings"
)

const comparedState = "_comparedState_"
const comparedResult = "_comparedResult_"

type TableRow map[string]string

type DbSyncConfig struct {
	Db1, Db2   string
	SyncTables []string
}

func main() {
	dbSyncConfig := readConfig()
	db1 := getDb(dbSyncConfig.Db1); defer db1.Close()
	db2 := getDb(dbSyncConfig.Db2); defer db2.Close()

	for _, tableName := range dbSyncConfig.SyncTables {
		sql := "select * from " + tableName
		rowsMap1 := execQuery(db1, sql)
		rowsMap2 := execQuery(db2, sql)

		compareRows(rowsMap1, rowsMap2)
		fmt.Print("Start to merge left " + tableName)
		rows1 := mergeRows(db1, tableName, rowsMap2)
		fmt.Printf(", merged %v rows\n", rows1)

		fmt.Print("Start to merge right " + tableName)
		rows2 := mergeRows(db2, tableName, rowsMap1)
		fmt.Printf(", merged %v rows\n", rows2)
	}
}

func readConfig() DbSyncConfig {
	fpath := "dbsync.toml"
	if len(os.Args) > 1 {
		fpath = os.Args[1]
	}

	dbSyncConfig := DbSyncConfig{}
	if _, err := toml.DecodeFile(fpath, &dbSyncConfig); err != nil {
		checkErr(err)
	}

	return dbSyncConfig
}

func mergeRows(db *sql.DB, tableName string, rows *map[string]TableRow) int {
	mergedRowsCount := 0
	for _, row := range *rows {
		// 一边有另外一边没有
		if row[comparedState] == "1" {
			// fmt.Println(row)
			mergedRowsCount += insertRow(db, tableName, &row)
		}
	}

	return mergedRowsCount
}

func insertRow(db *sql.DB, tableName string, row *TableRow) int {
	delete(*row, comparedState)
	delete(*row, comparedResult)

	sql, vals := compositeSql(tableName, row)
	stmt, err := db.Prepare(sql)
	checkErr(err)

	res, err := stmt.Exec(vals...)
	if err != nil {
		fmt.Println(vals); fmt.Println(err); return 0
	}

	rowCnt, err := res.RowsAffected()
	checkErr(err)

	return int(rowCnt)
}

func compositeSql(tableName string, row *TableRow) (string, []interface{}) {
	sql := "insert into " + tableName + "("
	vals := make([]interface{}, len(*row))

	i := 0; for key, val := range *row {
		if val == "NULL" {
			vals[i] = nil
		} else {
			vals[i] = val
		}
		sql += key + ","; i++
	}

	sql = strings.TrimRight(sql, ",") + ") values(" + strings.Repeat("?,", len(*row))
	sql = strings.TrimRight(sql, ",") + ")"
	// fmt.Println("vals", vals, "sql:", sql)

	return sql, vals
}

func compareRows(rows1, rows2 *map[string]TableRow) {
	for pk, row1 := range *rows1 {
		row2, ok := (*rows2)[pk]
		if !ok {
			row1[comparedState] = "1" /* 左边有右边没有*/; continue
		}

		// 两边都有, 比较内容
		deep_equal := reflect.DeepEqual(row1, row2)
		if deep_equal {
			row1[comparedState] = "2"; row2[comparedState] = "2"
		} else {
			msg := fmt.Sprintf("<<<%v\n>>>%v\n", row1, row2)
			row1[comparedState] = "3"; row2[comparedState] = "3"
			row1[comparedResult] = msg; row2[comparedResult] = msg
			fmt.Printf(msg)
		}
	}

	for _, row2 := range *rows2 {
		if _, ok := row2[comparedState]; !ok {
			row2[comparedState] = "1"
		}
	}
}

func execQuery(db *sql.DB, sql string) *map[string]TableRow {
	rows, err := db.Query(sql)
	checkErr(err)
	defer rows.Close()

	column, _ := rows.Columns()
	values := make([][]byte, len(column))
	scans := make([]interface{}, len(column))
	for i := range values {
		scans[i] = &values[i] //让每一行数据都填充到[][]byte里面
	}

	results := make(map[string]TableRow)

	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			checkErr(err)
		}

		row := make(TableRow)
		for k, v := range values {
			pk := column[k]
			if v == nil {
				row[pk] = "NULL"
			} else {
				row[pk] = string(v)
			}
		}

		primaryKey := string(values[0])
		results[primaryKey] = row
	}

	return &results
}

func getDb(dataSourceName string) *sql.DB {
	db, err := sql.Open("mysql", dataSourceName)
	checkErr(err)

	return db
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
