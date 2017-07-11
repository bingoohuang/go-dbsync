package main

import (
	"flag"
	"html/template"
	"net/http"
	"time"
	"os"
	"io"
	"log"
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"regexp"
)

type LogItem struct {
	LogName string
	LogFile string
}

var (
	port        *string
	contextPath string
	logItems    []LogItem
	homeTempl   = template.Must(template.New("").Parse(homeHTML))
	lineRegexp  *regexp.Regexp
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	port = flag.String("port", "8497", "tail log port number")
	logFlag := flag.String("log", "", "tail log file path")
	lineRegexpArg := flag.String("lineRegex",
		`^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`, "line regex") // 2017-07-11 18:07:01
	flag.Parse()
	contextPath = *contextPathArg
	lineRegexp = regexp.MustCompile(*lineRegexpArg)
	logItems = parseLogItems(*logFlag)
}

// go run src/go-tail-web.go -log=/Users/bingoo/gitlab/et-server/et.log -contextPath=/et
// go run src/go-tail-web.go -log=et:/Users/bingoo/gitlab/et-server/et.log,aa:aa.log -contextPath=/et
func main() {
	http.HandleFunc(contextPath+"/", serveHome)
	http.HandleFunc(contextPath+"/tail", serveTail)
	http.HandleFunc(contextPath+"/locate", serveLocate)
	if err := http.ListenAndServe(":" + *port, nil); err != nil {
		log.Fatal(err)
	}
}

func parseLogItems(logFlag string) []LogItem {
	logItems := splitTrim(logFlag, ",")

	result := make([]LogItem, 0)
	for i, logItem := range logItems {
		kvs := splitTrim(logItem, ":")

		logName := "LOG" + strconv.Itoa(i)
		logFile := kvs[0]

		if len(kvs) >= 2 {
			logName = kvs[0]
			logFile = kvs[1]
		}

		item := LogItem{
			logName,
			logFile,
		}

		result = append(result, item)
	}

	return result
}

func readFileIfModified(logFile string, lastMod time.Time, seekPos int64, filterKeyword string, initRead bool) ([]byte, time.Time, int64, error) {
	fi, err := os.Stat(logFile)
	if err != nil {
		return nil, lastMod, 0, err
	}
	if !fi.ModTime().After(lastMod) {
		return nil, lastMod, fi.Size(), nil
	}

	input, err := os.Open(logFile)
	if err != nil {
		return nil, lastMod, fi.Size(), err
	}
	defer input.Close()

	if seekPos < 0 {
		seekPos = fi.Size() + seekPos
	}

	if seekPos < 0 || seekPos > fi.Size() {
		seekPos = 0
	}

	if seekPos > 0 {
		if _, err := input.Seek(seekPos, 0); err != nil {
			return nil, lastMod, fi.Size(), err
		}
	}

	p, lastPos, err := readContent(input, seekPos, filterKeyword, initRead)
	return p, fi.ModTime(), lastPos, err
}

func containsAny(str string, sub []string) bool {
	if len(sub) == 0 {
		return true;
	}

	for _, v := range sub {
		if strings.Contains(str, v) {
			return true
		}
	}

	return false
}

func readContent(input io.ReadSeeker, startPos int64, filterKeyword string, initRead bool) ([]byte, int64, error) {
	subs := splitTrim(filterKeyword, ",")
	reader := bufio.NewReader(input)

	var buffer bytes.Buffer
	firstLine := startPos > 0 && initRead
	pos := startPos
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, pos, err
		}

		len := len(data)
		if len == 0 {
			break
		}

		pos += int64(len)
		if firstLine {
			// jump the first line because of it may be not full.
			firstLine = false
			continue
		}
		line := string(data)
		if containsAny(line, subs) {
			buffer.Write(data)
		}
	}

	return buffer.Bytes(), pos, nil
}

func splitTrim(str, sep string) []string {
	subs := strings.Split(str, sep)
	ret := make([]string, 0)
	for i, v := range subs {
		v := strings.TrimSpace(v)
		if len(subs[i]) > 0 {
			ret = append(ret, v)
		}
	}

	return ret
}

func hexString(val int64) string {
	return strconv.FormatInt(val, 16)
}

func parseHex(val string) (int64, error) {
	return strconv.ParseInt(val, 16, 64)
}

func findLogItem(logName string) *LogItem {
	for _, v := range logItems {
		if v.LogName == logName {
			return &v
		}
	}
	return nil
}

func serveLocate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	locateStart := strings.TrimSpace(req.FormValue("locateStart"))
	logName := req.FormValue("logName")
	if locateStart == "" {
		w.Write([]byte("locateStart should be non empty"))
		return
	}

	logFileName := findLogItem(logName).LogFile

	input, err := os.Open(logFileName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	defer input.Close()

	locateLines(input, locateStart, w)
}

func locateLines(input *os.File, locateStart string, w http.ResponseWriter) {
	reader := bufio.NewReader(input)
	locateStartFound := false
	prevLine := ""

	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				w.Write([]byte(err.Error()))
			}
			break
		}

		if len(data) == 0 {
			break
		}

		line := string(data)
		if strings.HasPrefix(line, locateStart) { // 找到了
			if !locateStartFound {
				w.Write([]byte(prevLine)) // 写入定位前面一行
				locateStartFound = true
			}
			w.Write(data)
		} else if locateStartFound { // 结束查找
			w.Write(data) // 非标准行，比如异常堆栈信息，或者写入定位下面一行
			if lineRegexp.MatchString(line) {
				break;
			}
		} else {
			prevLine = line
		}
	}
}

func serveTail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	n, err := parseHex(r.FormValue("lastMod"))
	if err != nil {
		w.Write([]byte("lastMod required"))
		return
	}

	lastMod := time.Unix(0, n)
	seekPos, err := parseHex(r.FormValue("seekPos"))
	filterKeyword := r.FormValue("filterKeyword")
	logName := r.FormValue("logName")
	logFileName := findLogItem(logName).LogFile

	p, lastMod, seekPos, err := readFileIfModified(logFileName, lastMod, seekPos, filterKeyword, false)
	if err != nil {
		log.Println("readFileIfModified error", err)
		return
	}

	w.Header().Set("last-mod", hexString(lastMod.UnixNano()))
	w.Header().Set("seek-pos", hexString(seekPos))
	w.Write(p)
}

type LogHomeItem struct {
	LogName string
	LogFile string
	Data    string
	SeekPos string
	LastMod string
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

	items := make([]LogHomeItem, len(logItems))

	for i, v := range (logItems) {
		p, lastMod, fileSize, err := readFileIfModified(v.LogFile, time.Time{},
			-6000, "", true)
		if err != nil {
			log.Println("readFileIfModified error", err)
			p = []byte(err.Error())
			lastMod = time.Unix(0, 0)
		}

		items[i] = LogHomeItem{
			v.LogName,
			v.LogFile,
			string(p),
			hexString(fileSize),
			hexString(lastMod.UnixNano()),
		}
	}

	v := struct {
		IsMoreThanOneLog bool
		LogItems    []LogHomeItem
	}{
		len(items) > 1,
		items,
	}
	err := homeTempl.Execute(w, &v)
	if err != nil {
		log.Println("template execute error", err)
		w.Write([]byte(err.Error()))
	}
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<title>log web</title>
<style>

{{if .IsMoreThanOneLog}}
	div.tab {
		overflow: hidden;
		position:fixed;
		bottom:0;
		font-size: 12px;
		background-color: #f1f1f1;
		left:0;
		right:0;
	}
	div.tab button {
		background-color: inherit;
		float: left;
		border: none;
		outline: none;
		cursor: pointer;
		padding: 10px 16px;
		transition: 0.3s;
	}

	div.tab button:hover {
		background-color: #ddd;
	}

	div.tab button.active {
		background-color: #ccc;
	}
{{end}}

.operateDiv {
	position:fixed;
	top:2px;
	left:8px;
	right:0;
	font-size: 12px;
	background-color: #f1f1f1;
}

.filterKeyword {
	width:300px;
}

pre {
	margin-top: 40px;
	margin-bottom: 50px;
	font-size: 10px;
}

.pre-wrap {
	white-space: pre-wrap;
}
button {
	padding:3px 10px;
}


.tabcontent {
{{if .IsMoreThanOneLog}}
	display: none;
{{end}}
	border-top: none;
}

</style>
<script src="https://cdn.bootcss.com/jquery/3.2.1/jquery.min.js"></script>
</head>
<body>

{{if .IsMoreThanOneLog}}
	<div class="tab">
		{{with .LogItems}}
		{{range .}}
		  <button class="tablinks">{{.LogName}}</button>
		{{end}}
		{{end}}
	</div>
{{end}}

{{with .LogItems}}
{{range .}}
<div id="{{.LogName}}" class="tabcontent">
	<pre class="fileDataPre">{{.Data}}</pre>
	<div class="operateDiv">
		<div>{{.LogFile}}</div>
		<input type="text" class="filterKeyword" placeholder="请输入过滤关键字"></input>
		<input type="checkbox" class="toggleWrapCheckbox">自动换行</input>
		<input type="checkbox" class="autoRefreshCheckbox">自动刷新</input>
		<button class="refreshButton">刷新</button>
		<button class="clearButton">清空</button>
		<button class="gotoBottomButton">直达底部</button>
		<input type="text" class="locateStart" placeholder="2017-10-07 18:50"></input>
		<button class="locateButton">定位</button>
		<input type="hidden" class="SeekPos" value="{{.SeekPos}}"/>
		<input type="hidden" class="LastMod" value="{{.LastMod}}"/>
	</div>
</div>
{{end}}
{{end}}

<script type="text/javascript">
(function() {
	var pathname = window.location.pathname
	if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
		pathname = pathname.substring(0, pathname.length - 1)
	}

{{if .IsMoreThanOneLog}}
	$('button.tablinks').click(function() {
		$('div.tabcontent').removeClass('active').hide()
		$('#' + $(this).text()).addClass('active').show()
	})
	$('button.tablinks').first().click()
{{end}}

	$('.clearButton').click(function() {
		var parent = $(this).parents('div.tabcontent')
		$('.fileDataPre', parent).empty()
	})

	var tailFunction = function(parent) {
		$.ajax({
			type: 'POST',
			url: pathname + "/tail",
			data: {
				seekPos: $('.SeekPos', parent).val(),
				lastMod: $('.LastMod', parent).val(),
				filterKeyword: $('.filterKeyword', parent).val(),
				logName: parent.prop('id')
			},
			success: function(content, textStatus, request){
				var seekPos = request.getResponseHeader('seek-pos')
				$('.SeekPos', parent).val(seekPos)
				var lastMod = request.getResponseHeader('last-mod')
				$('.LastMod', parent).val(lastMod)

				if (content != "" ) {
					$(".fileDataPre", parent).append(content)
					scrollToBottom()
				}
			},
			error: function (request, textStatus, errorThrown) {
				// alert("")
			}
		})
	}

	$('.refreshButton').click(function() {
		var parent = $(this).parents('div.tabcontent')
		tailFunction(parent)
	})

	var scrollToBottom = function() {
		$('html, body').scrollTop($(document).height())
	}

	var toggleWrapClick = function(parent) {
		var checked = $(".toggleWrapCheckbox", parent).is(':checked')
		$(".fileDataPre", parent).toggleClass("pre-wrap", checked)
		scrollToBottom()
	}

	$(".toggleWrapCheckbox").click(function() {
		var parent = $(this).parents('div.tabcontent')
		toggleWrapClick(parent)
	})

	var refreshTimer = {}

	var autoRefreshClick = function(parent) {
		var tabcontentId = parent.prop('id')
		if (refreshTimer[tabcontentId]) {
			clearInterval(refreshTimer[tabcontentId])
			refreshTimer[tabcontentId] = null
		}

		var checked = $(".autoRefreshCheckbox", parent).is(':checked')
		if (checked) {
			 refreshTimer[tabcontentId] = setInterval(function() {tailFunction(parent)}, 3000)
		}
		$('.refreshButton,.locateButton', parent).prop("disabled", checked);
	}

	$(".autoRefreshCheckbox").click(function() {
		var parent = $(this).parents('div.tabcontent')
		autoRefreshClick(parent)
	})

	scrollToBottom()

	$('.gotoBottomButton').click(scrollToBottom)

	$('.locateButton').click(function() {
		var parent = $(this).parents('div.tabcontent')
		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				locateStart: $('.locateStart', parent).val(),
				logName: parent.prop('id')
			},
			success: function(content, textStatus, request){
				if (content != "" ) {
					$(".fileDataPre", parent).text(content)
					scrollToBottom()
				} else {
					$(".fileDataPre", parent).text("empty content")
				}
			},
			error: function (request, textStatus, errorThrown) {
				// alert("")
			}
		})
	})
})()
</script>
</body>
</html>
`
