package mydb

import (
	"../myutil"
	"database/sql"
	"fmt"
	"strings"
)

type Db struct {
	db *sql.DB
}

func GetDb(dataSourceName string) *Db {
	db, err := sql.Open("mysql", dataSourceName)
	myutil.CheckErr(err)

	return &Db{db}
}

func (db *Db) Close() error {
	return db.db.Close()
}

func (db *Db) Query(sql string, args ...interface{}) *sql.Rows {
	rows, err := db.db.Query(sql, args...)
	myutil.CheckErr(err)
	return rows
}

func MakeColumnsValues(rows *sql.Rows) ([]string, [][]byte, []interface{}) {
	columns, _ := rows.Columns()
	values := make([][]byte, len(columns))
	scans := make([]interface{}, len(columns))
	for i := range values {
		scans[i] = &values[i] //让每一行数据都填充到[][]byte里面
	}

	return columns, values, scans
}

func ScanRow(rows *sql.Rows, columns []string, values [][]byte, scans []interface{}) map[string]string {
	if err := rows.Scan(scans...); err != nil {
		myutil.CheckErr(err)
	}

	row := make(map[string]string)
	for k, v := range values {
		col := columns[k]
		if v == nil {
			row[col] = "NULL"
		} else {
			row[col] = string(v)
		}
	}

	return row
}

func (db *Db) InsertRow(tableName string, row map[string]string) int {
	sql, vals := compositeSql(tableName, row)
	stmt, err := db.db.Prepare(sql)
	myutil.CheckErr(err)

	res, err := stmt.Exec(vals...)
	if err != nil {
		fmt.Println(vals)
		fmt.Println(err)
		return 0
	}

	rowCnt, err := res.RowsAffected()
	myutil.CheckErr(err)

	return int(rowCnt)
}

func compositeSql(tableName string, row map[string]string) (string, []interface{}) {
	mystr := myutil.MyStr{}
	mystr.PS("insert into ").PS(tableName).PS("(")
	vals := make([]interface{}, len(row))

	i := 0
	for key, val := range row {
		if val == "NULL" {
			vals[i] = nil
		} else {
			vals[i] = val
		}

		mystr.PS(key).PS(",")
		i++
	}

	mystr.ReplaceLast(")").PS(" values(").PS(strings.Repeat("?,", len(row))).ReplaceLast(")")
	sql := mystr.Str()
	// fmt.Println("vals", vals, "sql:", sql)

	return sql, vals
}
