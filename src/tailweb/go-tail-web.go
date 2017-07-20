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
	"time"
	"../myutil"
	"strconv"
)

var (
	port           *string
	contextPath    string
	logItems       []myutil.LogItem
	homeTempl      = template.Must(template.New("").Parse(homeHTML))
	lineRegexp     *regexp.Regexp
	tailMaxLines   int
	locateMaxLines int
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	port = flag.String("port", "8497", "tail log port number")
	logFlag := flag.String("log", "", "tail log file path")
	lineRegexpArg := flag.String("lineRegex",
		`^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`, "line regex") // 2017-07-11 18:07:01
	tailMaxLinesArg := flag.Int("tailMaxLines", 1000, "max lines per tail")
	locateMaxLinesArg := flag.Int("locateMaxLines", 500, "max lines per tail")

	flag.Parse()
	contextPath = *contextPathArg
	lineRegexp = regexp.MustCompile(*lineRegexpArg)
	logItems = myutil.ParseLogItems(*logFlag)
	tailMaxLines = *tailMaxLinesArg
	locateMaxLines = *locateMaxLinesArg
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

func readFileIfModified(logFile, filterKeywords string, lastMod time.Time, seekPos int64, initRead bool) ([]byte, time.Time, int64, error) {
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

	p, lastPos, err := readContent(input, seekPos, filterKeywords, initRead)
	return p, fi.ModTime(), lastPos, err
}

func readContent(input io.ReadSeeker, startPos int64, filterKeywords string, initRead bool) ([]byte, int64, error) {
	filters := myutil.SplitTrim(filterKeywords, ",")
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
	logName := req.FormValue("logName")
	filterKeywords := req.FormValue("filterKeywords")
	locateKeywords := req.FormValue("locateKeywords")
	pagingLog := req.FormValue("pagingLog")
	direction := req.FormValue("direction") // up or down
	findPos, err := strconv.ParseInt(req.FormValue("findPos"), 10, 64)

	if err != nil {
		http.Error(w, "findPos is illegal "+err.Error(), 405)
		return
	}

	logFileName := myutil.FindLogItem(logItems, logName).LogFile

	fi, err := os.Stat(logFileName)
	if err != nil {
		http.Error(w, "stat file "+err.Error(), 405)
		return
	}

	input, err := os.Open(logFileName)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	defer input.Close()

	filters := myutil.SplitTrim(filterKeywords, ",")
	locates := myutil.SplitTrim(locateKeywords, ",")

	if direction == "down" {
		if findPos > 0 && findPos < fi.Size() {
			if _, err := input.Seek(findPos, 0); err != nil {
				http.Error(w, "Seek error "+err.Error(), 405)
				return
			}
		}

		p, newPos := locateLines(input, findPos, pagingLog, filterKeywords, locateKeywords)
		w.Header().Set("Find-Pos", strconv.FormatInt(newPos, 10))
		w.Write(p)
	} else if direction == "up" {
		if pagingLog == "yes" {

		} else {
			locateStartFound, foundPos, err := locateBackTowardsStart(input, locates)
			if err != nil {
				http.Error(w, "locateBackTowardsStart¬ error "+err.Error(), 405)
				return
			}
			if !locateStartFound {
				w.Write([]byte("not found"))
				return
			} else {

			}
		}


		locateStartFound := false
		totalLines := 0
		var buffer *bytes.Buffer = nil
		readLines := 0;
		if findPos < 0 {
			findPos = fi.Size()
		}

	CONTINUE_READ:
		newStart := findPos
		newStart -= 6000
		if (newStart <= 0) {
			newStart = 0
		}
		if newStart >= findPos {
			w.Header().Set("Find-Pos", strconv.FormatInt(newStart, 10))
			if buffer != nil {
				w.Write([]byte("already at top"))
			} else {
				w.Write(buffer.Bytes())
			}
			return
		} else {
			if newStart > 0 && newStart < fi.Size() {
				if _, err := input.Seek(newStart, 0); err != nil {
					http.Error(w, "Seek error "+err.Error(), 405)
					return
				}
			}

		FIRST_FOUND_REREAD:
			p, lines, newReadLines, locateFound := locateLinesBackToward(input, newStart, findPos, pagingLog, filterKeywords, locateKeywords, locateStartFound, readLines)
			if pagingLog == "yes" || locateFound && locateStartFound {
				newBuffer := bytes.NewBuffer(p)
				if buffer != nil {
					newBuffer.Write(buffer.Bytes())
				}
				buffer = newBuffer

				totalLines += lines
				if totalLines >= locateMaxLines {
					w.Header().Set("Find-Pos", strconv.FormatInt(newStart, 10))
					w.Write(buffer.Bytes())
					return
				} else {
					goto CONTINUE_READ
				}
			} else {
				if locateFound {
					if !locateStartFound {
						readLines = newReadLines
						locateStartFound = true
						goto FIRST_FOUND_REREAD
					}
				} else {
					findPos = newStart
					goto CONTINUE_READ
				}
			}
		}
	}
}

func locateBackTowardsStart(input *os.File, locates []string) (bool, int64, error) {
	fi, err := input.Stat()
	if err != nil {
		return false, 0, err
	}

	if _, err := input.Seek(0, io.SeekEnd); err != nil {
		return false, 0, err
	}

	startPos := fi.Size()

	for {
		maxPos := startPos
		startPos = maxPos - 6000
		if startPos <= 0 {
			startPos = 0
		}

		if _, err := input.Seek(startPos-maxPos, io.SeekCurrent); err != nil {
			return false, maxPos, err
		}

		found, foundPos, err := locateBackTowardStart(input, startPos, maxPos, locates)
		if found {
			return true, foundPos, nil
		}
		if err != nil {
			return false, startPos, err
		}

		if startPos == 0 {
			return false, startPos, nil
		}
	}
}

func locateBackTowardStart(input *os.File, findPos, maxPos int64, locates []string) (bool, int64, error) {
	reader := bufio.NewReader(input)

	pos := findPos
	for pos < maxPos {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return false, pos, err
			}
			break
		}

		len := len(data)
		if len == 0 {
			break
		}

		pos += int64(len)

		line := string(data)
		if (myutil.ContainsAll(line, locates)) {
			return true, pos, nil
		}
	}

	return false, pos, nil
}

func readLines(input *os.File, startPos, endPos int64, leftLines int, filters []string) (content []byte,  readLines int) {
	input.Seek(startPos, io.SeekStart)
	reader := bufio.NewReader(input)

	readLines = 0
	pos := startPos
	var buffer bytes.Buffer
	firstLine := true
	for readLines < leftLines && pos <= endPos  {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				buffer.Write([]byte(err.Error()))
			}
			break
		}

		len := len(data)
		if len == 0 {
			break
		}
		if !firstLine {
			firstLine = false
			continue
		}

		pos += int64(len)

		line := string(data)
		if (myutil.ContainsAll(line, filters)) {
			buffer.WriteString(line)
			readLines++
		}
	}

	content = buffer.Bytes()
	return
}

func locateLines(input *os.File, findPos int64, pagingLog, filterKeywords, locateKeywords string) ([]byte, int64) {
	reader := bufio.NewReader(input)
	locateStartFound := false
	var prevLine []byte = nil

	filters := myutil.SplitTrim(filterKeywords, ",")
	locates := myutil.SplitTrim(locateKeywords, ",")
	lines := 0
	pos := findPos
	var buffer bytes.Buffer
	for lines < locateMaxLines {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				buffer.Write([]byte(err.Error()))
			}
			break
		}

		len := len(data)
		if len == 0 {
			break
		}

		pos += int64(len)

		line := string(data)
		if pagingLog == "yes" {
			if (myutil.ContainsAll(line, filters)) {
				buffer.WriteString(line)
			}
			lines++
		} else {
			if myutil.ContainsAll(line, locates) { // 包含关键字
				if !locateStartFound {
					buffer.Write([]byte(prevLine)) // 写入定位前面一行
					locateStartFound = true
				}
				buffer.Write(data)
				lines++
			} else if locateStartFound { // 结束查找
				if (myutil.ContainsAll(line, filters)) {
					buffer.WriteString(line)
				}
				lines++
			} else {
				prevLine = data
			}
		}
	}

	return buffer.Bytes(), pos
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
	filterKeywords := r.FormValue("filterKeywords")
	logName := r.FormValue("logName")
	logFileName := myutil.FindLogItem(logItems, logName).LogFile

	p, lastMod, seekPos, err := readFileIfModified(logFileName, filterKeywords, lastMod, seekPos, false)
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

.filterKeywords, .locateKeywords {
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

.locateDiv {
	margin-left: 50px;
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
		<span>
			<input type="text" class="filterKeywords" placeholder="请输入过滤关键字"></input>
			<input type="checkbox" class="toggleWrapCheckbox">自动换行</input>
			<input type="checkbox" class="autoRefreshCheckbox">自动刷新</input>
			<button class="refreshButton">刷新</button>
			<button class="clearButton">清空</button>
			<button class="gotoBottomButton">直达底部</button>
			<input type="hidden" class="SeekPos" value="{{$e.SeekPos}}"/>
			<input type="hidden" class="LastMod" value="{{$e.LastMod}}"/>
		</span>
		<span class="locateDiv">
			<input type="text" class="locateKeywords" placeholder="请输入查找关键字"></input>
			<button class="findFromBottom locateButton">从底向上找</button>
			<button class="findFromTop locateButton">从顶向下找</button>
			<nbsp/>
			<button class="prevPage locateButton">上一页</button>
			<button class="nextPage locateButton">下一页</button>
			<input type="hidden" class="findPos" value="-1"/>
		</span>
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
				filterKeywords: $('.filterKeywords', parent).val(),
				logName: parent.prop('id')
			},
			success: function(content, textStatus, request){
				$('.SeekPos', parent).val(request.getResponseHeader('Seek-Pos'))
				$('.LastMod', parent).val(request.getResponseHeader('Last-Mod'))

				if (content != "" ) {
					$(".fileDataPre", parent).append(content)
					scrollToBottom()
				}
			},
			error: function (request, textStatus, errorThrown) {
				alert(textStatus + ", " + errorThrown)
			}
		})
	}

	var pagingLog = function(parent, direction) {
		var tabcontentId = parent.prop('id')
		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				logName: parent.prop('id'),
				filterKeywords: $('.filterKeywords', parent).val(),
				locateKeywords: $('.locateKeywords', parent).val(),
				findPos: $('.findPos', parent).val(),
				direction: direction,
				pagingLog: 'yes'
			},
			success: function(content, textStatus, request){
				$('.findPos', parent).val(request.getResponseHeader('Find-Pos'))
				if (content == "" ) {
					alert("no more")
					return
				}

				var pre = $(".fileDataPre", parent)

				if (direction == 'down') {
					pre.append(content)
					scrollToBottom()
				} else if (direction == 'up') {
					pre.preppend(content)
					scrollToTop()
				}
			},
			error: function (request, textStatus, errorThrown) {
				alert(textStatus + ", " + errorThrown)
			}
		})
	}


	var locateLog = function(parent, direction) {
		var tabcontentId = parent.prop('id')
		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				logName: parent.prop('id'),
				filterKeywords: $('.filterKeywords', parent).val(),
				locateKeywords: $('.locateKeywords', parent).val(),
				findPos: $('.findPos', parent).val(),
				direction: direction,
				pagingLog: 'no'
			},
			success: function(content, textStatus, request){
				$('.findPos', parent).val(request.getResponseHeader('Find-Pos'))
				if (content == "" ) {
					alert("no more")
					return
				}

				var pre = $(".fileDataPre", parent)
				pre.text(content)

				if (direction == 'down') {
					scrollToBottom()
				} else if (direction == 'up') {
					scrollToTop()
				}
			},
			error: function (request, textStatus, errorThrown) {
				alert(textStatus + ", " + errorThrown)
			}
		})
	}


	$('.refreshButton').click(function() {
		var parent = $(this).parents('div.tabcontent')
		tailFunction(parent)
	})

	$('.nextPage').click(function() {
		var parent = $(this).parents('div.tabcontent')
		pagingLog(parent, 'down')
	})
	$('.prevPage').click(function() {
		var parent = $(this).parents('div.tabcontent')
		pagingLog(parent, 'up')
	})

	$('.findFromTop').click(function() {
		var parent = $(this).parents('div.tabcontent')

		$('.findPos', parent).val('0')
		locateLog(parent, 'down')
	})
	$('.findFromBottom').click(function() {
		var parent = $(this).parents('div.tabcontent')

		$('.findPos', parent).val('-1')
		locateLog(parent, 'up')
	})

	var scrollToBottom = function() {
		$('html, body').scrollTop($(document).height())
	}

	var scrollToTop = function() {
		$('html, body').scrollTop(0)
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
