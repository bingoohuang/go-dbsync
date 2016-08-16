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

type SyncParam struct {
	db1             *mydb.Db
	db2             *mydb.Db
	tableName       string
	nodb            *mynodb.Nodb
	rowChan1        chan map[string]string
	rowChan2        chan map[string]string
	diffRowsCount   int
	leftMergedRows  int
	rightMergedRows int
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
		rowChan2 := make(chan map[string]string)
		syncConf := SyncParam{db1, db2, tableName, nodb, rowChan1, rowChan2, 0, 0, 0}

		go syncConf.walkDb1()
		syncConf.mergeToDb2()

		go syncConf.walkDb2()
		syncConf.mergeToDb1()
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

func (syncParam *SyncParam) walkDb2() {
	rows := syncParam.db2.Query("select * from " + syncParam.tableName)
	defer rows.Close()
	defer close(syncParam.rowChan2)

	pkCol, _ := syncParam.nodb.Get(PK_COL)
	columns, values, scans := mydb.MakeColumnsValues(rows)

	for rows.Next() {
		row := mydb.ScanRow(rows, columns, values, scans)
		if !syncParam.nodb.Exists((*row)[pkCol]) {
			syncParam.rowChan2 <- *row
		}
	}
}

func (syncParam *SyncParam) walkDb1() {
	rows := syncParam.db1.Query("select * from " + syncParam.tableName)
	defer rows.Close()
	defer close(syncParam.rowChan1)

	columns, values, scans := mydb.MakeColumnsValues(rows)

	for rows.Next() {
		row := *mydb.ScanRow(rows, columns, values, scans)

		row[PK] = string(values[0])
		row[PK_COL] = columns[0]
		syncParam.rowChan1 <- row
	}
}

func (syncParam *SyncParam) mergeToDb1() {
	startTime := time.Now()
	fmt.Println("Start to merge left " + syncParam.tableName)
	for row1 := range syncParam.rowChan2 {
		syncParam.db1.InsertRow(syncParam.tableName, &row1)
		syncParam.leftMergedRows += 1
	}

	costTime := time.Now().Sub(startTime)
	fmt.Printf("Merged left %v with %v rows in %v\n",
		syncParam.tableName, syncParam.leftMergedRows, costTime)
}

func (syncParam *SyncParam) mergeToDb2() {
	startTime := time.Now()
	fmt.Println("Start to merge right " + syncParam.tableName)
	for row1 := range syncParam.rowChan1 {
		syncParam.mergeRowToDb2(row1)
	}

	costTime := time.Now().Sub(startTime)
	fmt.Printf("Merged right %v with %v rows in %v\n",
		syncParam.tableName, syncParam.rightMergedRows, costTime)
}

func (syncParam *SyncParam) mergeRowToDb2(row1 map[string]string) {
	pk := row1[PK]
	pkCol := row1[PK_COL]
	sql := "select * from " + syncParam.tableName + " where " + pkCol + " = ? limit 1"
	rows := syncParam.db2.Query(sql, pk)
	defer rows.Close()

	syncParam.nodb.Set(PK_COL, pkCol)
	delete(row1, PK)
	delete(row1, PK_COL)

	columns, values, scans := mydb.MakeColumnsValues(rows)
	if rows.Next() {
		row2 := mydb.ScanRow(rows, columns, values, scans)
		syncParam.compareRow(columns, &row1, row2)
	} else {
		syncParam.db2.InsertRow(syncParam.tableName, &row1)
		syncParam.nodb.Set(pk, "1")
		syncParam.rightMergedRows += 1
	}
}

func (syncParam *SyncParam) compareRow(columns []string, row1, row2 *map[string]string) {
	pk := (*row1)[PK]
	if reflect.DeepEqual(*row1, *row2) {
		syncParam.nodb.Set(pk, "2")
	}

	syncParam.nodb.Set(pk, "3")
	syncParam.diffRowsCount += 1

	msg := fmt.Sprintf("%v<<<%v\n%v>>>%v\n",
		syncParam.diffRowsCount+1, myutil.RowToString(columns, row1),
		syncParam.diffRowsCount+1, myutil.RowToString(columns, row2))
	fmt.Printf(msg)
}
