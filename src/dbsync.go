package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strings"
	"github.com/lunny/nodb"
	nodbConf "github.com/lunny/nodb/config"
	"reflect"
	"time"
	"io/ioutil"
	"bytes"
)

const PK = "_PK_"
const PK_COL = "_PK_COL_"

type TableRow map[string]string

type DbSyncConfig struct {
	Db1, Db2   string
	SyncTables []string
}

func main() {
	dbSyncConfig := readConfig()
	db1 := getDb(dbSyncConfig.Db1); defer db1.Close()
	db2 := getDb(dbSyncConfig.Db2); defer db2.Close()

	tempDir, tempNodb, _ := openTempNodb()
	defer os.RemoveAll(tempDir)

	for _, tableName := range dbSyncConfig.SyncTables {
		rowChan1 := make(chan TableRow)

		go walkDb1(rowChan1, db1, tableName)
		mergeToDb2(rowChan1, db2, tableName, tempNodb)

		rowChan2 := make(chan TableRow)
		go walkDb2(rowChan2, db2, tableName, tempNodb)
		mergeToDb1(rowChan2, db1, tableName)
	}
}

func SetNoDb(nodb *nodb.DB, key, value  string) error {
	return nodb.Set([]byte(key), []byte(value))
}

func GetNoDb(nodb *nodb.DB, key  string) (string, error) {
	value, err := nodb.Get([]byte(key))
	str := string(value)
	return str, err
}

func ExistsNoDb(nodb *nodb.DB, key  string) bool {
	value, _ := nodb.Exists([]byte(key))
	return value == 1
}

func openTempNodb() (string, *nodb.DB, error) {
	cfg := new(nodbConf.Config)

	cfg.DataDir, _ = ioutil.TempDir(os.TempDir(), "nodb")
	nodbs, err := nodb.Open(cfg)
	if err != nil {
		fmt.Printf("nodb: error opening db: %v", err)
	}

	db, err := nodbs.Select(0)

	return cfg.DataDir, db, err
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

func insertRow(db *sql.DB, tableName string, row *TableRow) int {
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

func walkDb2(rowChan chan <- TableRow, db *sql.DB, tableName string, nodb *nodb.DB) {
	sql := "select * from " + tableName
	rows, err := db.Query(sql)
	checkErr(err)
	defer rows.Close()

	columns, values, scans := makeColumnsValues(rows)

	pkCol, _ := GetNoDb(nodb, PK_COL);

	for rows.Next() {
		row := *scanRow(rows, columns, values, scans)
		if !ExistsNoDb(nodb, row[pkCol]) {
			rowChan <- row
		}
	}

	close(rowChan)
}

func walkDb1(rowChan chan <- TableRow, db *sql.DB, tableName string) {
	sql := "select * from " + tableName
	rows, err := db.Query(sql)
	checkErr(err)
	defer rows.Close()

	columns, values, scans := makeColumnsValues(rows)

	for rows.Next() {
		row := *scanRow(rows, columns, values, scans)

		row[PK] = string(values[0]); row[PK_COL] = columns[0]
		rowChan <- row
	}

	close(rowChan)
}

func makeColumnsValues(rows *sql.Rows) ([]string, [][]byte, []interface{}) {
	columns, _ := rows.Columns()
	values := make([][]byte, len(columns))
	scans := make([]interface{}, len(columns))
	for i := range values {
		scans[i] = &values[i] //让每一行数据都填充到[][]byte里面
	}

	return columns, values, scans
}

func scanRow(rows *sql.Rows, columns []string, values [][]byte, scans []interface{}) *TableRow {
	if err := rows.Scan(scans...); err != nil {
		checkErr(err)
	}

	row := make(TableRow)
	for k, v := range values {
		col := columns[k]
		if v == nil {
			row[col] = "NULL"
		} else {
			row[col] = string(v)
		}
	}

	return &row
}

func mergeToDb1(rowChan <-chan TableRow, db *sql.DB, tableName string) {
	startTime := time.Now()
	fmt.Println("Start to merge left " + tableName)
	mergedRowsCount := 0
	for row1 := range rowChan {
		mergedRowsCount += mergeRowToDb1(row1, db, tableName)
	}

	costTime := time.Now().Sub(startTime)
	fmt.Printf("Merged left %v with %v rows in %v\n", tableName, mergedRowsCount, costTime)

}

func mergeToDb2(rowChan <-chan TableRow, db *sql.DB, tableName string, nodb *nodb.DB) {
	startTime := time.Now()
	fmt.Println("Start to merge right " + tableName)
	mergedRowsCount, diffRowsCount := 0, 0
	for row1 := range rowChan {
		merged, diffs := mergeRowToDb2(diffRowsCount, row1, db, tableName, nodb)
		mergedRowsCount += merged
		diffRowsCount += diffs
	}

	costTime := time.Now().Sub(startTime)
	fmt.Printf("Merged right %v with %v rows in %v\n", tableName, mergedRowsCount, costTime)
}

func mergeRowToDb1(row1 TableRow, db *sql.DB, tableName string) int {
	insertRow(db, tableName, &row1)
	return 1
}

func mergeRowToDb2(diffRowsCount int, row1 TableRow, db *sql.DB, tableName string, nodb *nodb.DB) (int, int) {
	pk := row1[PK]; pkCol := row1[PK_COL]
	sql := "select * from " + tableName + " where " + pkCol + " = ? limit 1"
	rows, err := db.Query(sql, pk)
	checkErr(err)
	defer rows.Close()

	SetNoDb(nodb, PK_COL, pkCol)
	delete(row1, PK); delete(row1, PK_COL)

	columns, values, scans := makeColumnsValues(rows)
	if rows.Next() {
		row2 := *scanRow(rows, columns, values, scans)
		equal := compareRow(diffRowsCount, columns, &row1, &row2)
		if equal {
			SetNoDb(nodb, pk, "2")
			return 0, 0
		} else {
			SetNoDb(nodb, pk, "3")
			return 0, 1
		}
	} else {
		insertRow(db, tableName, &row1)
		SetNoDb(nodb, pk, "1")
		return 1, 0
	}
}

func compareRow(diffRowsCount int, columns []string, row1, row2 *TableRow) bool {
	deep_equal := reflect.DeepEqual(*row1, *row2)
	if deep_equal {
		return true
	}

	msg := fmt.Sprintf("%v<<<%v\n%v>>>%v\n", diffRowsCount + 1, strRow(columns, *row1),
		diffRowsCount + 1, strRow(columns, *row2))
	fmt.Printf(msg)
	return false
}

func strRow(cols []string, row TableRow) string {
	var rowStr bytes.Buffer
	rowStr.WriteString("{")
	for _, col := range cols {
		val, ok := row[col]
		if ok {
			writeKeyVal(rowStr, col, val)
			delete(row, col)
		}
	}

	for key, val := range row {
		writeKeyVal(rowStr, key, val)
	}

	rowStr.WriteString("}")

	return rowStr.String()
}

func writeKeyVal(buf *bytes.Buffer, key, val string) {
	if buf.Len() > 1 {
		buf.WriteString(", ")
	}

	buf.WriteString(key);
	buf.WriteString(":");
	buf.WriteString(val);
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
