package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/xwb1989/sqlparser"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	contextPath       string
	homeTempl         = template.Must(template.New("").Parse(homeHTML))
	port              int
	maxRows           int
	dataSource        string
	writeAuthRequired bool
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 8381, "Port to serve.")
	maxRowsArg := flag.Int("maxRows", 1000, "Max number of rows to return.")
	dataSourceArg := flag.String("dataSource", "user:pass@tcp(127.0.0.1:3306)/db?charset=utf8", "dataSource string.")
	writeAuthRequiredArg := flag.Bool("writeAuthRequired", true, "write auth required")

	flag.Parse()

	contextPath = *contextPathArg
	port = *portArg
	maxRows = *maxRowsArg
	dataSource = *dataSourceArg
	writeAuthRequired = *writeAuthRequiredArg
}

func main() {
	http.HandleFunc(contextPath+"/", serveHome)
	http.HandleFunc(contextPath+"/query", serveQuery)
	http.HandleFunc(contextPath+"/searchDb", serveSearchDb)
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

type SqlHistory struct {
	SqlTime string
	Sql     string
}

func saveHistory(sql string) {
	sqlHistory := SqlHistory{
		time.Now().Format("2006-01-02 15:04:05.000"),
		sql,
	}
	json, _ := json.Marshal(sqlHistory)
	file, _ := os.OpenFile("sqlHistory.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	file.Write(json)
	file.WriteString("\n")
	file.Close()
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

	if writeAuthRequired {
		start := time.Now()
		sqlParseResult, _ := sqlparser.Parse(querySql)

		switch sqlParseResult.(type) {
		case *sqlparser.Insert, *sqlparser.Delete, *sqlparser.Update, *sqlparser.Set:
			json.NewEncoder(w).Encode(QueryResult{Headers: nil, Rows: nil,
				Error:         "dangerous sql, please get authorized first!",
				ExecutionTime: start.Format("2006-01-02 15:04:05.000"),
				CostTime:      time.Since(start).String(),
			})
			log.Println("sql", querySql, "is not allowed because of insert/delete/update/set")
			return
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
func showHistory() (header []string, data [][]string, executionTime, costTime string, err error) {
	header = nil
	data = nil
	start := time.Now()
	executionTime = start.Format("2006-01-02 15:04:05.000")

	file, err := os.OpenFile("sqlHistory.json", os.O_RDONLY, 0660)
	if err != nil {
		costTime = time.Since(start).String()
		err = errors.New("no history")
		return
	}
	defer file.Close()

	header = []string{"ExecutionTime", "Sql"}
	data = make([][]string, 0)

	reader := bufio.NewReader(file)
	rowIndex := 0
	for {
		rowData, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}

		len := len(rowData)
		if len == 0 {
			break
		}

		rowIndex++
		var sqlHistory SqlHistory
		json.Unmarshal(rowData, &sqlHistory)
		row := []string{strconv.Itoa(rowIndex), sqlHistory.SqlTime, sqlHistory.Sql}

		data = append(data, row)
	}

	costTime = time.Since(start).String()
	err = nil
	return
}

func selectDb(tid string) (string, error) {
	queryDbSql := "SELECT DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME FROM TR_F_DB WHERE MERCHANT_ID = '" + tid + "' LIMIT 1"

	_, data, _, _, err := executeQuery(queryDbSql, dataSource)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "", errors.New("no db found")
	} else if len(data) > 1 {
		return "", errors.New("more than one db found")
	}

	row := data[0]

	// user:pass@tcp(127.0.0.1:3306)/db?charset=utf8
	return row[1] + ":" + row[2] + "@tcp(" + row[3] + ":" + row[4] + ")/" + row[5] + "?charset=utf8mb4,utf8&timeout=3s", nil
}

func executeQuery(querySql, dataSource string) ([]string /*header*/, [][]string /*data*/, string /*executionTime*/, string /*costTime*/, error) {
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, nil, "", "", err
	}
	defer db.Close()

	header, data, executionTime, costTime, err := query(db, querySql, maxRows)
	return header, data, executionTime, costTime, err
}

func query(db *sql.DB, query string, maxRows int) ([]string, [][]string, string, string, error) {
	log.Printf("querying: %s", query)
	start := time.Now()
	executionTime := start.Format("2006-01-02 15:04:05.000")
	rows, err := db.Query(query)

	costTime := time.Since(start).String()
	if err != nil {
		return nil, nil, executionTime, costTime, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, executionTime, costTime, err
	}

	columnSize := len(columns)

	data := make([][]string, 0)

	for row := 1; rows.Next() && row <= maxRows; row++ {
		strValues := make([]sql.NullString, columnSize+1)
		strValues[0] = sql.NullString{strconv.Itoa(row), true}
		pointers := make([]interface{}, columnSize)
		for i := 0; i < columnSize; i++ {
			pointers[i] = &strValues[i+1]
		}
		if err := rows.Scan(pointers...); err != nil {
			return columns, data, executionTime, "", err
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
	return columns, data, executionTime, costTime, nil
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
<head> <title>sql web</title>
<style>
button { padding:3px 10px; }
table { width: 100%; border-collapse: collapse; }
table td { border: 1px solid #eeeeee; white-space: nowrap; }
.error { color: red; }
.searchKey { width: 150px; }
.searchResult span { border: 1px solid #ccc; cursor: pointer; margin-right: 10px; border-radius: 10px; }
.searchResult .active { background-color: #ccc; font-weight:bold; }
.CodeMirror { border-top: 1px solid #f7f7f7; border-bottom: 1px solid #f7f7f7;}
.tables span {float: left; width: 20%; text-decoration: underline; color: blue; padding: 0 5px; cursor: pointer; user-select: none;}
.result {clear: both; padding-top: 10px;}
table.executionSummary tr:first-child td { background-color: aliceblue; }
table.queryResult tr:first-child td { background-color: #f7f7f7; }
.wrapper {width:100%; max-height:155px; overflow:auto; border: 1px solid #f7f7f7; background-color: #f7f7f7;}
.handle { background: #f7f7f7; height: 20px; user-select: none; cursor: row-resize; border-top: 1px solid #f7f7f7; border-bottom: 1px solid #f7f7f7; }
.handle:before { content: '\2261'; /* https://en.wikipedia.org/wiki/Triple_bar */ color: #999; position: absolute; left: 50%; }
.handle:hover { background: #f0f0f0; }
.handle:hover:before { color: #000; }
</style>
<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.2.1/jquery.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.28.0/codemirror.min.js"></script>
<script src="https://unpkg.com/sql-formatter@2.0.0/dist/sql-formatter.min.js"></script>
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.28.0/codemirror.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.28.0/mode/sql/sql.min.js"></script>
</head>
<body>
<div>
	<input type="text" placeholder="tid/tcode/name" class="searchKey">
	<button class="searchButton">Find DB</button>
	<span class="searchResult"></span>
</div>
<div>
	<textarea  class="sql" id="code" cols="120" rows="5">-- input sql here</textarea>
	<div class="handle"> </div>
	<button class="executeQuery">Run SQL</button>
	<button class="formatSql">Format SQL</button>
&nbsp;
	<button class="clearSql">Clear SQL</button>
	<button class="collapseSql">Collapse SQL</button>
	<button class="clearResult">Clear Result</button>
</div>
<div class="wrapper">
	<div class="tables"></div>
</div>
<div class="result"></div>
<script>
(function() {
	var MIN_HEIGHT = 60
	var start_y
	var start_h

	function on_drag(e) {
		var newHeight = Math.max(MIN_HEIGHT, (start_h + e.y - start_y)) + "px"
		codeMirror.setSize(null, newHeight)
	}
	function on_release(e) {
		document.body.removeEventListener("mousemove", on_drag);
		window.removeEventListener("mouseup", on_release);
	}

	$('.handle')[0].addEventListener("mousedown", function (e) {
		start_y = e.y
		start_h = $('.CodeMirror').height()
		document.body.addEventListener("mousemove", on_drag)
		window.addEventListener("mouseup", on_release)
	})

	var mac = CodeMirror.keyMap.default == CodeMirror.keyMap.macDefault // 判断是否为Mac
	var runKey = (mac ? "Cmd" : "Ctrl") + "-Enter"
	var extraKeys = {}
	extraKeys[runKey] = function(cm) {
		var executeQuery = $('.executeQuery')
		if (!executeQuery.prop("disabled")) executeQuery.click()
	}

	var codeMirror = CodeMirror.fromTextArea(document.getElementById('code'), {
		mode: 'text/x-mysql',
		indentWithTabs: true,
		smartIndent: true,
		lineNumbers: true,
		matchBrackets : true,
		extraKeys: extraKeys
	})
	codeMirror.setSize(null, '60px')

	$('.collapseSql').click(function(){
		codeMirror.setSize(null, '60px')
	});

	var pathname = window.location.pathname
	if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
		pathname = pathname.substring(0, pathname.length - 1)
	}

	var executeSql = function(sql) {
		$.ajax({
			type: 'POST',
			url: pathname + "/query",
			data: { tid: activeMerchantId, sql: sql },
			success: function(content, textStatus, request){
				tableCreate(content, sql)
			}
		})
	}

	$('.executeQuery').prop("disabled", true).click(function() {
		var sql = codeMirror.somethingSelected() ? codeMirror.getSelection() : codeMirror.getValue()
		executeSql(sql)
	})

	function tableCreate(result, sql) {
		var table = '<table class="executionSummary"><tr><td>time</td><td>cost</td><td>sql</td><td>error</td></tr>'
		+ '<tr><td>' + result.ExecutionTime  + '</td><td>' + result.CostTime  + '</td><td>' + sql + '</td><td'
	    + (result.Error && (' class="error">' + result.Error) || '>OK')
		+ '</td><tr></table><br/><table class="queryResult">'

		if (result.Headers && result.Headers.length > 0 ) {
			table += '<tr><td>#</td><td>' + result.Headers.join('</td><td>') + '</td></tr>'
		}
		if (result.Rows && result.Rows.length > 0 ) {
			for (var i = 0; i < result.Rows.length; i++) {
				table += '<tr><td>' + result.Rows[i].join('</td><td>') + '</td></tr>'
			}
		}
		table += '</table><br/>'
		$(table).prependTo($('.result'))
	}

	$('.clearResult').click(function() {
		$('.result').html('')
	})

	$('.searchKey').keydown(function(event){
		var keyCode =  event.keyCode  || event.which
		if (keyCode == 13) $('.searchButton').click()
	})

	$('.searchButton').click(function() {
		$.ajax({
			type: 'POST',
			url: pathname + "/searchDb",
			data: { searchKey: $('.searchKey').val() },
			success: function(content, textStatus, request){
				var searchResult = $('.searchResult')
				var searchHtml = ''
				if (content && content.length ) {
					for (var j = 0; j <  content.length; j++) {
						searchHtml += '<span tid="' + content[j].MerchantId + '">🌀' + content[j].MerchantName + '</span>'
					}
				} else {
					$('.executeQuery').prop("disabled", true)
					$('.tables').html('')
				}
				searchResult.html(searchHtml)
				$('.searchResult span:first-child').click()
			}
		})
	})

	var showTables = function(result) {
		var resultHtml = ''
		if (result.Rows && result.Rows.length > 0 ) {
			for (var i = 0; i < result.Rows.length; i++) {
				resultHtml += '<span>' + result.Rows[i][1] + '</span>'
			}
		}
		$('.tables').html(resultHtml)
	}

	var showTablesAjax = function(activeMerchantId) {
		$.ajax({
			type: 'POST',
			url: pathname + "/query",
			data: {tid: activeMerchantId, sql: 'show tables'},
			success: function(content, textStatus, request){
				showTables(content)
			}
		})
	}

	$('.tables').on('click', 'span', function(event) {
		var $button = $(this);
		var tableName = $(this).text()
		if ($button.data('alreadyclicked')) {
			$button.data('alreadyclicked', false) // reset
			if ($button.data('alreadyclickedTimeout')){
				clearTimeout($button.data('alreadyclickedTimeout')); // prevent this from happening
			}
			executeSql('show full columns from ' + tableName)
		} else {
			$button.data('alreadyclicked', true);
			var alreadyclickedTimeout=setTimeout(function(){
				$button.data('alreadyclicked', false); // reset when it happens
				executeSql('select * from '+ tableName)
			},300); // <-- dblclick tolerance here
			$button.data('alreadyclickedTimeout', alreadyclickedTimeout); // store this id to clear if necessary
		}
		return false;
	})

	var activeMerchantId = null
	$('.searchResult').on('click', 'span', function() {
		$('.searchResult span').removeClass('active')
		$(this).addClass('active')
		activeMerchantId = $(this).attr('tid')
		$('.executeQuery').prop("disabled", false)
		showTablesAjax(activeMerchantId)
	})

	$('.formatSql').click(function() {
		var sql = codeMirror.somethingSelected() ? codeMirror.getSelection() : codeMirror.getValue()
		var formattedSql = sqlFormatter.format(sql, {language: 'sql'})
		codeMirror.setValue(formattedSql)
	})
	$('.clearSql').click(function() {
		codeMirror.setValue('')
	})
})()
</script>
</body>
</html>
`
