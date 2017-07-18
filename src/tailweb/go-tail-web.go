package main

import (
	"bufio"
	"bytes"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"../myutil"
)

var (
	port         *string
	contextPath  string
	logItems     []myutil.LogItem
	homeTempl    = template.Must(template.New("").Parse(homeHTML))
	lineRegexp   *regexp.Regexp
	tailMaxLines int
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	port = flag.String("port", "8497", "tail log port number")
	logFlag := flag.String("log", "", "tail log file path")
	lineRegexpArg := flag.String("lineRegex",
		`^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`, "line regex") // 2017-07-11 18:07:01
	tailMaxLinesArg := flag.Int("tailMaxLines", 1000, "max lines per tail")

	flag.Parse()
	contextPath = *contextPathArg
	lineRegexp = regexp.MustCompile(*lineRegexpArg)
	logItems = myutil.ParseLogItems(*logFlag)
	tailMaxLines = *tailMaxLinesArg
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

func readFileIfModified(logFile, filterKeyword string, lastMod time.Time, seekPos int64, initRead bool) ([]byte, time.Time, int64, error) {
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
			return nil, lastMod, seekPos, err
		}
	}

	p, lastPos, err := readContent(input, seekPos, filterKeyword, initRead)
	return p, fi.ModTime(), lastPos, err
}

func readContent(input io.ReadSeeker, startPos int64, filterKeyword string, initRead bool) ([]byte, int64, error) {
	filters := myutil.SplitTrim(filterKeyword, ",")
	reader := bufio.NewReader(input)

	var buffer bytes.Buffer
	firstLine := startPos > 0 && initRead
	pos := startPos
	lastContains := false
	writtenLines := 0
	for writtenLines < tailMaxLines {
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
		if myutil.ContainsAll(line, filters) { // 包含关键字，直接写入
			buffer.Write(data)
			writtenLines++
			lastContains = true
		} else if lastContains { // 上次包含
			if lineRegexp.MatchString(line) { // 完整的日志行开始
				lastContains = false
			} else { // 本次是多行中的其他行
				buffer.Write(data)
				writtenLines++
			}
		}
	}

	return buffer.Bytes(), pos, nil
}

func serveLocate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	locateStart := strings.TrimSpace(req.FormValue("locateStart"))
	logName := req.FormValue("logName")
	filterKeyword := req.FormValue("filterKeyword")

	if locateStart == "" {
		w.Write([]byte("locateStart should be non empty"))
		return
	}

	logFileName := myutil.FindLogItem(logItems, logName).LogFile

	input, err := os.Open(logFileName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	defer input.Close()

	locateLines(input, locateStart, filterKeyword, w)
}

func locateLines(input *os.File, locateStart, filterKeyword string, w http.ResponseWriter) {
	reader := bufio.NewReader(input)
	locateStartFound := false
	prevLine := ""

	filters := myutil.SplitTrim(filterKeyword, ",")
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
			if myutil.ContainsAll(line, filters) { // 包含关键字
				w.Write(data)
			}
		} else if locateStartFound { // 结束查找
			w.Write(data) // 非标准行，比如异常堆栈信息，或者写入定位下面一行
			if lineRegexp.MatchString(line) {
				break
			}
		} else {
			prevLine = line
		}
	}
}

func serveTail(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	n, err := myutil.ParseHex(r.FormValue("lastMod"))
	if err != nil {
		http.Error(w, "lastMod required", 405)
		return
	}

	lastMod := time.Unix(0, n)
	seekPos, err := myutil.ParseHex(r.FormValue("seekPos"))
	filterKeyword := r.FormValue("filterKeyword")
	logName := r.FormValue("logName")
	logFileName := myutil.FindLogItem(logItems, logName).LogFile

	p, lastMod, seekPos, err := readFileIfModified(logFileName, filterKeyword, lastMod, seekPos, false)
	if err != nil {
		http.Error(w, string(err.Error()), 405)
		return
	}

	header.Set("Last-Mod", myutil.HexString(lastMod.UnixNano()))
	header.Set("Seek-Pos", myutil.HexString(seekPos))
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

	now := time.Time{}
	for i, v := range logItems {
		p, lastMod, fileSize, err := readFileIfModified(v.LogFile, "", now, -6000, true)
		if err != nil {
			log.Println("readFileIfModified error", err)
			p = []byte(err.Error())
			lastMod = time.Unix(0, 0)
		}

		items[i] = LogHomeItem{
			v.LogName,
			v.LogFile,
			string(p),
			myutil.HexString(fileSize),
			myutil.HexString(lastMod.UnixNano()),
		}
	}

	v := struct {
		IsMoreThanOneLog bool
		LogItems         []LogHomeItem
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
		font-weight:bold;
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

{{range $i, $e := .LogItems}}
<div id="{{$e.LogName}}" class="tabcontent">
	<pre class="fileDataPre">{{$e.Data}}</pre>
	<div class="operateDiv">
		<div>{{$e.LogFile}}</div>
		<input type="text" class="filterKeyword" placeholder="请输入过滤关键字"></input>
		<input type="checkbox" class="toggleWrapCheckbox">自动换行</input>
		<input type="checkbox" class="autoRefreshCheckbox">自动刷新</input>
		<button class="refreshButton">刷新</button>
		<button class="clearButton">清空</button>
		<button class="gotoBottomButton">直达底部</button>
		<input type="text" class="locateStart" placeholder="2017-10-07 18:50"></input>
		<button class="locateButton">定位</button>
		<input type="hidden" class="SeekPos" value="{{$e.SeekPos}}"/>
		<input type="hidden" class="LastMod" value="{{$e.LastMod}}"/>
	</div>
</div>
{{end}}

<script type="text/javascript">
(function() {
	var pathname = window.location.pathname
	if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
		pathname = pathname.substring(0, pathname.length - 1)
	}

{{if .IsMoreThanOneLog}}
	$('button.tablinks').click(function() {
		$('button.tablinks').removeClass('active')
		$(this).addClass('active')
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
				$('.SeekPos', parent).val(request.getResponseHeader('Seek-Pos'))
				$('.LastMod', parent).val(request.getResponseHeader('Last-Mod'))

				if (content != "" ) {
					$(".fileDataPre", parent).append(content)
					scrollToBottom()
				}
			}
		})
	}

	$('.locateButton').click(function() {
		var parent = $(this).parents('div.tabcontent')
		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				locateStart: $('.locateStart', parent).val(),
				logName: parent.prop('id'),
				filterKeyword: $('.filterKeyword', parent).val()
			},
			success: function(content, textStatus, request){
				if (content != "" ) {
					$(".fileDataPre", parent).text(content)
					scrollToBottom()
				} else {
					$(".fileDataPre", parent).text("empty content")
				}
			}
		})
	})

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
})()
</script>
</body>
</html>
`
