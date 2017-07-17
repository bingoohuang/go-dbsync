package main

import (
	"flag"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"net/http"
	"html/template"
	"strings"
	"encoding/json"
)

var (
	contextPath string
	homeTempl   = template.Must(template.New("").Parse(homeHTML))
	port        int
	maxRows     int
	dataSource  string
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 8080, "Port to serve.")
	maxRowsArg := flag.Int("maxRows", 50, "Max number of rows to return.")
	dataSourceArg := flag.String("dataSource", "user:pass@tcp(127.0.0.1:3306)/db?charset=utf8", "Max number of rows to return.")

	flag.Parse()

	contextPath = *contextPathArg
	port = *portArg
	maxRows = *maxRowsArg
	dataSource = *dataSourceArg
}

func main() {
	http.HandleFunc(contextPath+"/", serveHome)
	http.HandleFunc(contextPath+"/query", serveQuery)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}

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

	v := struct {
		Something bool
	}{
		false,
	}
	err := homeTempl.Execute(w, &v)
	if err != nil {
		log.Println("template execute error", err)
		w.Write([]byte(err.Error()))
	}
}

type QueryResult struct {
	Headers []string
	Rows    [][]string
	Error   string
}

func serveQuery(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	querySql := strings.TrimSpace(req.FormValue("sql"))

	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()

	header, data, err := query(db, querySql, maxRows)
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	queryResult := QueryResult{
		Headers: header,
		Rows:    data,
		Error:   errMsg,
	}

	json.NewEncoder(w).Encode(queryResult)
}

func query(db *sql.DB, query string, maxRows int) ([]string, [][]string, error) {
	log.Printf("querying: %s", query)
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	columnSize := len(columns)

	data := make([][]string, 0)

	for row := 1; rows.Next() && row < maxRows; row++ {
		strValues := make([]string, columnSize+1)
		strValues[0] = strconv.Itoa(row)
		pointers := make([]interface{}, columnSize)
		for i := 0; i < columnSize; i++ {
			pointers[i] = &strValues[i+1]
		}
		if err := rows.Scan(pointers...); err != nil {
			return columns, data, err
		}
		data = append(data, strValues)
	}
	return columns, data, nil
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<title>sql web</title>
<style>

button {
	padding:3px 10px;
}

.sql {
	width:100%;
}

table {
	border-collapse: collapse;
	border: 1px solid;
}

tr {
	border: 0;
	margin: 0;
}

th,td {
	border: 1px solid;
}

.error {
	color: red;
}

</style>
<script src="https://cdn.bootcss.com/jquery/3.2.1/jquery.min.js"></script>
</head>
<body>
<div>
<input type="textarea" class="sql" placeholder="请输入SQL"></input>
<button class="executeQuery">刷新</button>
</div>
<br/>
<div class="result"></div>
<script>
(function() {
	var pathname = window.location.pathname
	if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
		pathname = pathname.substring(0, pathname.length - 1)
	}
	$('.executeQuery').click(function() {
		var sql = $('.sql').val()
		$.ajax({
			type: 'POST',
			url: pathname + "/query",
			data: {
				sql: sql
			},
			success: function(content, textStatus, request){
				tableCreate(content, sql)
			}
		})
	})

	function tableCreate(result, sql) {
		if (result.Error != "") {
			$('<div class="executedSql">' + sql + '</div><div class="error">' + result.Error + '</div>').appendTo($('.result'))
			return
		}

		var table = '<div class="executedSql">' + sql + '</div><table><thead><tr>'
		table += '<th>#</th>'
		for (var i = 0; i < result.Headers.length; i++) {
			table += '<th>' +  result.Headers[i] + '</th>'
		}

		table += '</tr></thead><tbody>'

		for (var i = 0; i < result.Rows.length; i++) {
			table += '<tr>'
			for (var j = 0; j <  result.Rows[i].length; j++) {
				table += '<td>' + result.Rows[i][j] + '</td>'
			}
			table += '</tr>'
		}
		table += '</tbody></table>'
		$( table).appendTo($('.result'))
	}
})()
</script>
</body>
</html>
`
