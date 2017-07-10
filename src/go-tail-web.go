package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"time"
	"os"
	"io"
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

var (
	port        = flag.String("port", "8497", "tail log port number")
	logFileName = flag.String("log", "", "tail log file path")
	homeTempl   = template.Must(template.New("").Parse(homeHTML))
)

func readFileIfModified(lastMod time.Time, seekPos, endPos int64, filterKeyword string) ([]byte, time.Time, int64, error) {
	fi, err := os.Stat(*logFileName)
	if err != nil {
		return nil, lastMod, 0, err
	}
	if !fi.ModTime().After(lastMod) {
		return nil, lastMod, fi.Size(), nil
	}

	input, err := os.Open(*logFileName)
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

	if _, err := input.Seek(seekPos, 0); err != nil {
		return nil, lastMod, fi.Size(), err
	}

	p, lastPos, err := readContent(input, seekPos, endPos, filterKeyword)
	return p, fi.ModTime(), lastPos, err
}

func containsAny(str string, sub []string) bool {
	for _, v := range sub {
		if strings.Contains(str, v) {
			return true
		}
	}

	return false
}

func readContent(input io.ReadSeeker, startPos, endPos int64, filterKeyword string) ([]byte, int64, error) {
	subs := splitTrim(filterKeyword)

	reader := bufio.NewReader(input)

	var buffer bytes.Buffer
	firstLine := true
	pos := startPos
	for endPos < 0 || pos < endPos {
		data, err := reader.ReadBytes('\n')
		len := len(data)
		pos += int64(len)
		if err == nil || err == io.EOF {
			if firstLine { // jump the first line because of it may be not full.
				firstLine = false
				continue
			}
			if len > 0 {
				line := string(data)
				if filterKeyword == "" || containsAny(line, subs) {
					buffer.WriteString(line)
				}
			} else {
				break
			}
		} else if err != nil {
			if err == io.EOF {
				break
			}
			return nil, pos, err
		}
	}

	return buffer.Bytes(), pos, nil
}
func splitTrim(filterKeyword string) []string {
	subs := strings.Split(filterKeyword, ",")
	for i, v := range subs {
		subs[i] = strings.TrimSpace(v)
	}
	return subs
}

func hexString(val int64) string {
	return strconv.FormatInt(val, 16)
}

func parseHex(val string) (int64, error) {
	return strconv.ParseInt(val, 16, 64)
}

func serveLocate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	locateStart := strings.TrimSpace(req.FormValue("locateStart"))
	if locateStart == "" {
		w.Write([]byte("locateStart should be non empty"))
		return
	}

	input, err := os.Open(*logFileName)
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
	lastLine := ""
	for {
		data, err := reader.ReadBytes('\n')
		if err == nil || err == io.EOF {
			if len(data) > 0 {
				line := string(data)
				if strings.HasPrefix(line, locateStart) { // 找到了
					if !locateStartFound {
						w.Write([]byte(lastLine))
						locateStartFound = true
					}
					w.Write(data)
				} else if locateStartFound { // 结束查找
					w.Write(data)
					break;
				} else {
					lastLine = line
				}
			} else {
				break
			}
		} else if err != nil {
			if err != io.EOF {
				w.Write([]byte(err.Error()))
			}
			break
		}
	}
}

func serveTail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var lastMod time.Time
	if n, err := parseHex(r.FormValue("lastMod")); err == nil {
		lastMod = time.Unix(0, n)
	}

	seekPos, err := parseHex(r.FormValue("seekPos"))

	filterKeyword := r.FormValue("filterKeyword")

	p, lastMod, seekPos, err := readFileIfModified(lastMod, seekPos, -1, filterKeyword)
	if err != nil {
		log.Println("readFileIfModified error", err)
		return
	}

	w.Header().Set("last-mod", hexString(lastMod.UnixNano()))
	w.Header().Set("seek-pos", hexString(seekPos))
	w.Write(p)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	p, lastMod, fileSize, err := readFileIfModified(time.Time{}, -6000, -1, "")
	if err != nil {
		log.Println("readFileIfModified error", err)
		p = []byte(err.Error())
		lastMod = time.Unix(0, 0)
	}

	var v = struct {
		Data        string
		SeekPos     string
		LastMod     string
		LogFileName string
	}{
		string(p),
		hexString(fileSize),
		hexString(lastMod.UnixNano()),
		*logFileName,
	}
	homeTempl.Execute(w, &v)
}

func main() {
	flag.Parse()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/tail", serveTail)
	http.HandleFunc("/locate", serveLocate)
	if err := http.ListenAndServe(":" + *port, nil); err != nil {
		log.Fatal(err)
	}
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<title>{{.LogFileName}}</title>
<style>
#filterKeyword {
	width:300px;
}
.pre-wrap {
	 white-space: pre-wrap;
}
button {
	padding:3px 50px;
}
</style>
<script src="https://cdn.bootcss.com/jquery/3.2.1/jquery.min.js"></script>
</head>
<body>
	<pre id="fileDataPre">{{.Data}}</pre>
	<input type="text" id="filterKeyword" placeholder="请输入过滤关键字"></input>
	<input type="checkbox" id="toggleWrapCheckbox">自动换行</input>
	<input type="checkbox" id="autoRefreshCheckbox">自动刷新</input>
	<button id="refreshButton">刷新</button>
	<button id="clearButton">清空</button>
	<input type="text" id="locateStart" placeholder="2017-10-07 18:50"></input>
	<button id="locateButton">定位</button>
<script type="text/javascript">
(function() {
	var seekPos = "{{.SeekPos}}"
	var lastMod = "{{.LastMod}}"
	var pathname = window.location.pathname
	if (pathname == "/") {
		pathname = ""
	}

	$('#clearButton').click(function() {
		$('#fileDataPre').empty()
	})

	var tailFunction = function() {
		$.ajax({
			type: 'POST',
			url: pathname + "/tail",
			data: {
				seekPos: seekPos,
				lastMod: lastMod,
				filterKeyword: $('#filterKeyword').val()
			},
			success: function(content, textStatus, request){
				seekPos = request.getResponseHeader('seek-pos')
				lastMod = request.getResponseHeader('last-mod')
				if (content != "" ) {
					$("#fileDataPre").append(content)
					scrollToBottom()
				}
			},
			error: function (request, textStatus, errorThrown) {
				// alert("")
			}
		})
	}

	$('#refreshButton').click(tailFunction)

	var scrollToBottom = function() {
		$('html, body').scrollTop($(document).height())
	}

	var toggleWrapClick = function() {
		var checked = $("#toggleWrapCheckbox").is(':checked')
		$("#fileDataPre").toggleClass("pre-wrap", checked)
		scrollToBottom()
	}
	$("#toggleWrapCheckbox").click(toggleWrapClick)
	toggleWrapClick()

	var refreshTimer = null
	var autoRefreshClick = function() {
		if (refreshTimer != null) {
			clearInterval(refreshTimer)
			refreshTimer = null
		}

		var checked = $("#autoRefreshCheckbox").is(':checked')
		if (checked) {
			 refreshTimer = setInterval(tailFunction, 3000)
		}
	}
	$("#autoRefreshCheckbox").click(autoRefreshClick)
	autoRefreshClick()

	scrollToBottom()

	$('#locateButton').click(function() {
		$.ajax({
			type: 'POST',
			url: pathname + "/locate",
			data: {
				locateStart: $('#locateStart').val()
			},
			success: function(content, textStatus, request){
				if (content != "" ) {
					$("#fileDataPre").text(content)
					scrollToBottom()
				} else {
					$("#fileDataPre").text("empty content")
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
