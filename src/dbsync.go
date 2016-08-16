package main

import (
	"./mydb"
	"./mynodb"
	"./myutil"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"reflect"
	"time"
)

const PK = "_PK_"
const PK_COL = "_PK_COL_"

type DbSyncConfig struct {
	Db1, Db2   string
	SyncTables []string
}

func main() {
	dbSyncConfig := readConfig()
	db1 := mydb.GetDb(dbSyncConfig.Db1)
	defer db1.Close()
	db2 := mydb.GetDb(dbSyncConfig.Db2)
	defer db2.Close()

	nodb, tempDir, _ := mynodb.OpenTemp()
	defer os.RemoveAll(tempDir)

	for _, tableName := range dbSyncConfig.SyncTables {
		rowChan1 := make(chan map[string]string)
		go walkDb1(rowChan1, db1, tableName)
		mergeToDb2(rowChan1, db2, tableName, nodb)

		rowChan2 := make(chan map[string]string)
		go walkDb2(rowChan2, db2, tableName, nodb)
		mergeToDb1(rowChan2, db1, tableName)
	}
}

func readConfig() DbSyncConfig {
	fpath := "dbsync.toml"
	if len(os.Args) > 1 {
		fpath = os.Args[1]
	}

	dbSyncConfig := DbSyncConfig{}
	if _, err := toml.DecodeFile(fpath, &dbSyncConfig); err != nil {
		myutil.CheckErr(err)
	}

	return dbSyncConfig
}

func walkDb2(rowChan chan<- map[string]string, db *mydb.Db, tableName string, nodb *mynodb.Nodb) {
	rows := db.Query("select * from " + tableName)
	defer rows.Close()
	defer close(rowChan)

	pkCol, _ := nodb.Get(PK_COL)
	columns, values, scans := mydb.MakeColumnsValues(rows)

	for rows.Next() {
		row := *mydb.ScanRow(rows, columns, values, scans)
		if !nodb.Exists(row[pkCol]) {
			rowChan <- row
		}
	}
}

func walkDb1(rowChan chan<- map[string]string, db *mydb.Db, tableName string) {
	rows := db.Query("select * from " + tableName)
	defer rows.Close()
	defer close(rowChan)

	columns, values, scans := mydb.MakeColumnsValues(rows)

	for rows.Next() {
		row := *mydb.ScanRow(rows, columns, values, scans)

		row[PK] = string(values[0])
		row[PK_COL] = columns[0]
		rowChan <- row
	}
}

func mergeToDb1(rowChan <-chan map[string]string, db *mydb.Db, tableName string) {
	startTime := time.Now()
	fmt.Println("Start to merge left " + tableName)
	mergedRowsCount := 0
	for row1 := range rowChan {
		db.InsertRow(tableName, &row1)
		mergedRowsCount += 1
	}

	costTime := time.Now().Sub(startTime)
	fmt.Printf("Merged left %v with %v rows in %v\n", tableName, mergedRowsCount, costTime)

}

func mergeToDb2(rowChan <-chan map[string]string, db *mydb.Db, tableName string, nodb *mynodb.Nodb) {
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

func mergeRowToDb2(diffRowsCount int, row1 map[string]string, db *mydb.Db, tableName string, nodb *mynodb.Nodb) (int, int) {
	pk := row1[PK]
	pkCol := row1[PK_COL]
	sql := "select * from " + tableName + " where " + pkCol + " = ? limit 1"
	rows := db.Query(sql, pk)
	defer rows.Close()

	nodb.Set(PK_COL, pkCol)
	delete(row1, PK)
	delete(row1, PK_COL)

	columns, values, scans := mydb.MakeColumnsValues(rows)
	if rows.Next() {
		row2 := *mydb.ScanRow(rows, columns, values, scans)
		equal := compareRow(diffRowsCount, columns, &row1, &row2)
		if equal {
			nodb.Set(pk, "2")
			return 0, 0
		} else {
			nodb.Set(pk, "3")
			return 0, 1
		}
	} else {
		db.InsertRow(tableName, &row1)
		nodb.Set(pk, "1")
		return 1, 0
	}
}

func compareRow(diffRowsCount int, columns []string, row1, row2 *map[string]string) bool {
	if reflect.DeepEqual(*row1, *row2) {
		return true
	}

	msg := fmt.Sprintf("%v<<<%v\n%v>>>%v\n",
		diffRowsCount+1, myutil.RowToString(columns, row1),
		diffRowsCount+1, myutil.RowToString(columns, row2))
	fmt.Printf(msg)
	return false
}
