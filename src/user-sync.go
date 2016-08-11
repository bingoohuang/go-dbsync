package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"fmt"
	"reflect"
	"os"
)

type User struct {
	userId, mobile, openid sql.NullString
	merchantId, createTime sql.NullString
	effective              sql.NullString
	comparedState          uint8
	comparedResult         *string
}

// user-sync "root:mypw@tcp(localhost:13306)/dba" "root:mypw@tcp(localhost:13306)/dbb"
func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./user-sync dataSourceName1 dataSourceName2\n" +
			"Example:./user-sync \"root:mypw@tcp(localhost:3306)/dba\" \"root:mypw@tcp(localhost:3306)/dbb\"")
		return
	}

	dataSourceName1 := os.Args[1]
	db1 := getDb(dataSourceName1)
	defer db1.Close()

	dataSourceName2 := os.Args[2]
	db2 := getDb(dataSourceName2)
	defer db2.Close()

	users1 := queryRows(db1)
	mapUsers1 := makeMap(users1)
	users2 := queryRows(db2)
	mapUsers2 := makeMap(users2)

	compare(mapUsers1, mapUsers2)

	rows1 := merge(db1, mapUsers2)
	rows2 := merge(db2, mapUsers1)
	fmt.Printf("<<< Merged %v rows, >>> Merged %v rows", rows1, rows2)
}

func merge(db *sql.DB, users map[string]*User) int {
	rows := 0
	for _, user := range users {
		if user.comparedState == 1 {
			// 一边有另外一边没有
			rows += insertUser(db, user)
		}
	}

	return rows
}

func compare(users1 map[string]*User, users2 map[string]*User) {
	for userId, user1 := range users1 {
		user2, ok := users2[userId]
		if !ok {
			user1.comparedState = 1 // 左边有右边没有
			continue
		}

		// 两边都有, 比较内容
		deep_equal := reflect.DeepEqual(user1, user2)
		if deep_equal {
			user1.comparedState = 2
			user2.comparedState = 2
		} else {
			user1.comparedState = 3
			user2.comparedState = 3
			comparedResult := fmt.Sprintf("<<<\n%s\n>>>\n%s\n", strUser(user1), strUser(user2))
			user1.comparedResult = &comparedResult
			user2.comparedResult = user1.comparedResult
			fmt.Printf(comparedResult)
		}
	}

	for _, user2 := range users2 {
		if user2.comparedState == 0 {
			user2.comparedState = 1
		}
	}
}

func strUser(u *User) string {
	return nullStr(u.userId) + "," + nullStr(u.mobile) +
		"," + nullStr(u.openid) + "," + nullStr(u.merchantId) +
		"," + nullStr(u.createTime) + "," + nullStr(u.effective)
}

func nullStr(str sql.NullString) string {
	if str.Valid {
		return str.String
	} else {
		return "NULL"
	}

}

func makeMap(users []*User) map[string]*User {
	mapa := make(map[string]*User)
	for _, v := range users {
		mapa[v.userId.String] = v
	}

	return mapa
}

func printRows(users []*User) {
	for _, v := range users {
		fmt.Println(v)
	}
}

func printMap(users map[string]*User) {
	for k, v := range users {
		fmt.Println("k:", k, "v:", v)
	}
}

func getDb(dataSourceName string) *sql.DB {
	db, err := sql.Open("mysql", dataSourceName)
	checkErr(err)

	return db
}

func insertUser(db *sql.DB, user *User) int {
	stmt, err := db.Prepare("insert into tr_f_user(" +
		"user_id, mobile, openid, merchant_id, create_time, effective) " +
		"values(?, ?, ?, ?, ?, ?)")
	checkErr(err)

	res, err := stmt.Exec(user.userId, user.mobile, user.openid,
		user.merchantId, user.createTime, user.effective)
	checkErr(err)
	rowCnt, err := res.RowsAffected()
	checkErr(err)

	return int(rowCnt)
}

func queryRows(db *sql.DB) (users []*User) {
	rows, err := db.Query("select user_id, mobile, openid, " +
		"merchant_id, create_time, effective from tr_f_user")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		row := new(User)

		err := rows.Scan(&row.userId, &row.mobile, &row.openid,
			&row.merchantId, &row.createTime, &row.effective)
		checkErr(err)

		users = append(users, row)
	}
	err = rows.Err()
	checkErr(err)

	return
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
