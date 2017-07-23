package main

import (
	"../myutil"
	"bufio"
	"bytes"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	blockSize = 3000
)

var (
	port           *string
	contextPath    string
	logItems       []myutil.LogItem
	homeTempl      = template.Must(template.New("").Parse(homeHTML))
	tailMaxLines   int
	locateMaxLines int
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	port = flag.String("port", "8497", "tail log port number")
	logFlag := flag.String("log", "", "tail log file path")
	tailMaxLinesArg := flag.Int("tailMaxLines", 300, "max lines per tail")
	locateMaxLinesArg := flag.Int("locateMaxLines", 100, "max lines per tail")

	flag.Parse()
	contextPath = *contextPathArg
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
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
}

func readFileIfModified(logFile string, filters []string, lastMod time.Time, seekPos int64, initRead bool) ([]byte, time.Time, int64, error) {
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
		if _, err := input.Seek(seekPos, io.SeekStart); err != nil {
			return nil, lastMod, seekPos, err
		}
	}

	p, lastPos, err := readContent(input, seekPos, filters, initRead)
	return p, fi.ModTime(), lastPos, err
}

func readContent(input io.ReadSeeker, startPos int64, filters []string, initRead bool) ([]byte, int64, error) {
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
			if myutil.StartWithBlank(line) { // 本次是多行中的其他行
				buffer.Write(data)
				writtenLines++
			} else { // 完整的日志行开始
				lastContains = false
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
	startPos, _ := strconv.ParseInt(req.FormValue("startPos"), 10, 64)
	endPos, _ := strconv.ParseInt(req.FormValue("endPos"), 10, 64)

	logFileName := myutil.FindLogItem(logItems, logName).LogFile

	log.Println("locate logFileName:", logFileName, ", startPos:", startPos, ", endPos:", endPos,
		",direction:", direction, ",pagingLog:", pagingLog,
		",locateKeywords:", locateKeywords, ",filterKeywords", filterKeywords)

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
	fileSize := fi.Size()

	log.Println("log file size:", fileSize)

	if direction == "down" {
		if endPos < 0 {
			endPos = fileSize
		}
		if endPos > 0 {
			if endPos < fileSize {
				endPos -= 1
				input.Seek(endPos, io.SeekStart)
			} else {
				response(w, startPos, endPos, []byte("already reached bottom.\n"))
				return
			}
		}

		if pagingLog == "yes" {
			p, _, readEndPos, _ := readLines(input, endPos, -1, true, locateMaxLines, filters)
			response(w, startPos, readEndPos, p)
		} else {
			found, foundPos, foundLine, err := locateForwardStart(input, endPos, locates)
			log.Println("locate found", found, ", at pos", foundPos, ",err:", err)
			if err != nil {
				response(w, startPos, endPos, []byte("locateForwardsStart¬ error "+err.Error()+"\n"))
			} else if !found {
				response(w, startPos, endPos, []byte("not found"))
			} else {
				w.Write([]byte(foundLine))
				p, _, newPos, _ := readLines(input, foundPos, -1, false, locateMaxLines, filters)
				response(w, foundPos, newPos, p)
			}
		}
	} else if direction == "up" {
		if startPos < 0 {
			startPos = fileSize
			log.Println("change startPos to ", startPos)
		}

		if pagingLog == "yes" {
			if startPos > 0 {
				p, newPos := readUpLinesUntilMax(input, startPos, locateMaxLines, filters)
				log.Println("paging up result int startPos:", newPos, ", endPos:", startPos)
				response(w, newPos, startPos, p)
			} else {
				response(w, startPos, endPos, []byte("already reached top.\n"))
			}
		} else {
			found, foundPos, err := locateBackwardsStart(input, startPos, fileSize, locates)
			log.Println("locate found", found, ", at pos", foundPos, ",err:", err)
			if err != nil {
				response(w, startPos, endPos, []byte("locateBackwardsStart¬ error "+err.Error()+"\n"))
			} else if !found {
				response(w, endPos, endPos, []byte("not found"))
			} else {
				p, newPos := readUpLinesUntilMax(input, foundPos, locateMaxLines, filters)
				response(w, newPos, foundPos, p)
			}
		}
	}
}

func response(w http.ResponseWriter, startPos, endPos int64, content []byte) {
	log.Println("response with startPos", startPos, " and endPos", endPos)
	w.Header().Set("Start-Pos", strconv.FormatInt(startPos, 10))
	w.Header().Set("End-Pos", strconv.FormatInt(endPos, 10))
	w.Write(content)
}

func locateForwardStart(input *os.File, startPos int64, locates []string) (found bool, foundPos int64, foundLine string, err error) {
	if len(locates) == 0 {
		return true, startPos, "", nil
	}

	reader := bufio.NewReader(input)

	pos := startPos
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return false, pos, "", err
			}
			break
		}

		len := len(data)
		if len == 0 {
			break
		}

		line := string(data)
		if myutil.ContainsAll(line, locates) {
			return true, pos, line, nil
		}

		pos += int64(len)
	}

	return false, pos, "", nil
}

func locateBackwardsStart(input *os.File, startPos, fileSize int64, locates []string) (bool, int64, error) {
	if len(locates) == 0 {
		return true, fileSize, nil
	}

	if _, err := input.Seek(startPos, io.SeekStart); err != nil {
		return false, 0, err
	}

	var stepBack int64 = blockSize

	for {
		maxPos := startPos
		startPos = maxPos - stepBack
		if startPos <= 0 {
			startPos = 0
		}

		if _, err := input.Seek(startPos, io.SeekStart); err != nil {
			return false, maxPos, err
		}

		found, foundPos, resetFindPos, err := locateBackwardStartBlock(input, startPos, maxPos, locates)
		if found {
			return true, foundPos, nil
		}
		if err != nil {
			return false, resetFindPos, err
		}

		if startPos == 0 {
			return false, resetFindPos, nil
		}

		stepBack = resetFindPos - startPos + blockSize
	}
}

func locateBackwardStartBlock(input *os.File, findPos, maxPos int64, locates []string) (bool, int64, int64, error) {
	reader := bufio.NewReader(input)
	firstLine := true

	resetFindPos := findPos
	pos := findPos
	for pos < maxPos {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return false, pos, resetFindPos, err
			}
			break
		}

		len := len(data)
		if len == 0 {
			break
		}

		pos += int64(len)

		if firstLine {
			resetFindPos = pos
			firstLine = false
			continue
		}

		line := string(data)
		if myutil.ContainsAll(line, locates) {
			return true, pos, resetFindPos, nil
		}
	}

	return false, pos, resetFindPos, nil
}

func readUpLinesUntilMax(input *os.File, endPos int64, leftLines int, filters []string) (content []byte, newPos int64) {
	buffer := bytes.NewBuffer(make([]byte, 0))

	var stepBack int64 = blockSize
	for endPos >= 0 && leftLines > 0 {
		newStart := endPos - stepBack
		if newStart < 0 {
			newStart = 0
		}
		if newStart > 0 {
			newStart -= 1 // for jump first break line
		}

		input.Seek(newStart, io.SeekStart)
		p, newStartPos, _, lines := readLines(input, newStart, endPos, true, leftLines, filters)
		leftLines -= lines
		log.Println("read from ", newStart, ", got lines ", lines, " with newStartPos ", newStartPos, ", now left lines", leftLines)

		stepBack = newStartPos - newStart + blockSize

		if lines > 0 {
			pb := bytes.NewBuffer(p)
			pb.Write(buffer.Bytes())
			buffer = pb
		}

		endPos = newStartPos
		if newStart == 0 {
			break
		}
	}

	return buffer.Bytes(), endPos
}

// jumpFirstLine 第一行可能不完整，需要跳过
func readLines(input *os.File, startPos, endPos int64, jumpFirstLine bool, leftLines int, filters []string) (content []byte, newStartPos, readEndPos int64, linesRead int) {
	reader := bufio.NewReader(input)

	linesRead = 0
	readEndPos = startPos
	var buffer bytes.Buffer
	firstLine := jumpFirstLine && startPos > 0

	for linesRead < leftLines && (endPos < 0 || readEndPos < endPos) {
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
		readEndPos += int64(len)

		if firstLine {
			firstLine = false
			newStartPos = readEndPos
			continue
		}

		line := string(data)
		if myutil.ContainsAll(line, filters) {
			buffer.Write(data)
			linesRead++
		}
	}

	content = buffer.Bytes()
	return
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

	filters := myutil.SplitTrim(filterKeywords, ",")
	p, lastMod, seekPos, err := readFileIfModified(logFileName, filters, lastMod, seekPos, false)
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
	emptyStringArray := make([]string, 0)

	for i, v := range logItems {
		p, lastMod, fileSize, err := readFileIfModified(v.LogFile, emptyStringArray, now, -blockSize, true)
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
			<button class="gotoBottomButton">探底</button>
			<button class="gotoTopButton">登顶</button>
			<input type="hidden" class="SeekPos" value="{{$e.SeekPos}}"/>
			<input type="hidden" class="LastMod" value="{{$e.LastMod}}"/>
		</span>
		<span class="locateDiv">
			<input type="text" class="locateKeywords" placeholder="请输入查找关键字"></input>
			<button class="findFromBottom locateButton">从底向上找</button>
			<button class="findUp locateButton">继续向上找</button>
			<button class="findFromTop locateButton">从顶向下找</button>
			<button class="findDown locateButton">继续向下找</button>
			<nbsp/>
			<button class="prevPage locateButton">上一页</button>
			<button class="nextPage locateButton">下一页</button>
			<input type="hidden" class="StartPos" value="-1"/>
			<input type="hidden" class="EndPos" value="-1"/>
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

	var requesting = function(parent, disabled){
		$('.refreshButton,.locateButton', parent).prop("disabled", disabled);
	}

	var pagingLog = function(parent, direction) {
		requesting(parent, true)

		var tabcontentId = parent.prop('id')
		var startPos = $('.StartPos', parent).val()
		var endPos = $('.EndPos', parent).val()

		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				logName: parent.prop('id'),
				filterKeywords: $('.filterKeywords', parent).val(),
				locateKeywords: $('.locateKeywords', parent).val(),
				startPos: startPos,
				endPos: endPos,
				direction: direction,
				pagingLog: 'yes'
			},
			success: function(content, textStatus, request){
				$('.StartPos', parent).val(request.getResponseHeader('Start-Pos'))
				$('.EndPos', parent).val(request.getResponseHeader('End-Pos'))

				var pre = $(".fileDataPre", parent)

				if (startPos == -1 && endPos == -1) {
					pre.text(content)
				} else if (direction == 'down') {
					pre.append(content)
				} else if (direction == 'up') {
					pre.prepend(content)
				}

				if (direction == 'down') {
					scrollToBottom()
				} else if (direction == 'up') {
					scrollToTop()
				}
				requesting(parent, false)
			},
			error: function (request, textStatus, errorThrown) {
				alert(textStatus + ", " + errorThrown)
				requesting(parent, false)
			}
		})
	}


	var locateLog = function(parent, direction) {
		requesting(parent, true)
		var tabcontentId = parent.prop('id')
		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				logName: parent.prop('id'),
				filterKeywords: $('.filterKeywords', parent).val(),
				locateKeywords: $('.locateKeywords', parent).val(),
				startPos: $('.StartPos', parent).val(),
				endPos: $('.EndPos', parent).val(),
				direction: direction,
				pagingLog: 'no'
			},
			success: function(content, textStatus, request){
				$('.StartPos', parent).val(request.getResponseHeader('Start-Pos'))
				$('.EndPos', parent).val(request.getResponseHeader('End-Pos'))

				var pre = $(".fileDataPre", parent)
				pre.text(content)

				if (direction == 'down') {
					scrollToBottom()
				} else if (direction == 'up') {
					scrollToTop()
				}
				requesting(parent, false)
			},
			error: function (request, textStatus, errorThrown) {
				alert(textStatus + ", " + errorThrown)
				requesting(parent, false)
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

	$('.findUp').click(function() {
		var parent = $(this).parents('div.tabcontent')
		locateLog(parent, 'up')
	})
	$('.findDown').click(function() {
		var parent = $(this).parents('div.tabcontent')
		locateLog(parent, 'down')
	})

	$('.findFromTop').click(function() {
		var parent = $(this).parents('div.tabcontent')

		$('.StartPos', parent).val('0')
		$('.EndPos', parent).val('0')
		locateLog(parent, 'down')
	})
	$('.findFromBottom').click(function() {
		var parent = $(this).parents('div.tabcontent')

		$('.StartPos', parent).val('-1')
		$('.EndPos', parent).val('-1')
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
	$('.gotoTopButton').click(scrollToTop)
})()
</script>
</body>
</html>
`
