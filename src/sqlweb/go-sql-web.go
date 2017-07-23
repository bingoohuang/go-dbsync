package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	Headers       []string
	Rows          [][]string
	Error         string
	ExecutionTime string
	CostTime      string
}

func serveQuery(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	querySql := strings.TrimSpace(req.FormValue("sql"))

	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()

	header, data, err, executionTime, costTime := query(db, querySql, maxRows)
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	queryResult := QueryResult{
		Headers:       header,
		Rows:          data,
		Error:         errMsg,
		ExecutionTime: executionTime,
		CostTime:      costTime,
	}

	json.NewEncoder(w).Encode(queryResult)
}

func query(db *sql.DB, query string, maxRows int) ([]string, [][]string, error, string, string) {
	log.Printf("querying: %s", query)
	start := time.Now()
	executionTime := start.Format("2006-01-02 15:04:05.000")
	rows, err := db.Query(query)

	costTime := time.Since(start).String()
	if err != nil {
		return nil, nil, err, executionTime, costTime
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err, executionTime, costTime
	}

	columnSize := len(columns)

	data := make([][]string, 0)

	for row := 1; rows.Next() && row < maxRows; row++ {
		strValues := make([]sql.NullString, columnSize+1)
		strValues[0] = sql.NullString{strconv.Itoa(row), true}
		pointers := make([]interface{}, columnSize)
		for i := 0; i < columnSize; i++ {
			pointers[i] = &strValues[i+1]
		}
		if err := rows.Scan(pointers...); err != nil {
			return columns, data, err, executionTime, ""
		}

		values := make([]string, columnSize+1)
		for i, v := range strValues {
			if v.Valid {
				values[i] = v.String
			} else {
				values[i] = "(null)"
			}
		}

		data = append(data, values)
	}

	costTime = time.Since(start).String()
	return columns, data, nil, executionTime, costTime
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
	width:60%;
}
table {
  width: 100%;
  border-collapse: collapse;
}
table td {
  border: 1px solid #eeeeee;
  white-space: nowrap;
}
.error {
	color: red;
}
.CodeMirror {
    border-top: 1px solid black;
    border-bottom: 1px solid black;
}
</style>
<script src="https://cdn.bootcss.com/jquery/3.2.1/jquery.min.js"></script>
<script src="http://codemirror.net/1/js/codemirror.js" type="text/javascript"></script>

</head>
<body>
<div style="border-top: 1px solid black; border-bottom: 1px solid black;">
<textarea  class="sql" id="code" cols="120" rows="5">
SELECT NOW()
</textarea>
<button class="executeQuery">Execute</button>
</div>
<script type="text/javascript">

</script>
<br/>
<div class="result"></div>
<script>

(function() {
	var codeMirror = CodeMirror.fromTextArea('code', {
		height: "60px",
		parserfile: "http://codemirror.net/1/contrib/sql/js/parsesql.js",
		stylesheet: "http://codemirror.net/1/contrib/sql/css/sqlcolors.css",
		path: "http://codemirror.net/1/js/",
		textWrapping: true
	})
	var pathname = window.location.pathname
	if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
		pathname = pathname.substring(0, pathname.length - 1)
	}
	$('.executeQuery').click(function() {
		var sql = codeMirror.getCode()
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
		var table = ''

		table += '<table><tr><td>time</td><td>cost</td><td>sql</td><td>error</td></tr>'
		+ '<tr><td>' + result.ExecutionTime  + '</td><td>' + result.CostTime  + '</td><td>' + sql + '</td><td>'
		+ (result.Error || 'OK') + '</td><tr></table><br/>'

		table += '<table>'
		if (result.Headers && result.Headers.length > 0 ) {
			table += '<tr><td>#</td>'
			for (var i = 0; i < result.Headers.length; i++) {
				table += '<td>' + i + ":" +  result.Headers[i] + '</td>'
			}

			table += '</tr>'
		}

		if (result.Rows && result.Rows.length > 0 ) {
			for (var i = 0; i < result.Rows.length; i++) {
				table += '<tr>'
				for (var j = 0; j <  result.Rows[i].length; j++) {
					table += '<td>' + result.Rows[i][j] + '</td>'
				}
				table += '</tr>'
			}
		}
		table += '</table>'


		$(table).prependTo($('.result'))
	}
})()
</script>
</body>
</html>
`
