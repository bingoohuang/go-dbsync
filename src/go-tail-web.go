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
)

var (
	port        = flag.String("port", "8497", "tail log port number")
	logFileName = flag.String("log", "", "tail log file path")
	homeTempl   = template.Must(template.New("").Parse(homeHTML))
)

func readFileIfModified(lastMod time.Time, seekPos, endPos int64) ([]byte, time.Time, int64, error) {
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

	if _, err := input.Seek(seekPos, 0); err != nil {
		return nil, lastMod, fi.Size(), err
	}

	p, lastPos, err := readContent(input, seekPos, endPos)
	return p, fi.ModTime(), lastPos, err
}

func readContent(input io.ReadSeeker, startPos, endPos int64) ([]byte, int64, error) {
	r := bufio.NewReader(input)

	var buffer bytes.Buffer
	firstLine := true
	pos := startPos
	for endPos < 0 || pos < endPos {
		data, err := r.ReadBytes('\n')
		len := len(data)
		pos += int64(len)
		if err == nil || err == io.EOF {
			if firstLine { // jump the first line because of it may be not full.
				firstLine = false
				continue;
			}
			if len > 0 {
				buffer.Write(data)
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

func hexString(val int64) string {
	return strconv.FormatInt(val, 16)
}

func parseHex(val string) (int64, error) {
	return strconv.ParseInt(val, 16, 64)
}

func serveTail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var lastMod time.Time
	if n, err := parseHex(r.FormValue("lastMod")); err == nil {
		lastMod = time.Unix(0, n)
	}

	seekPos, err := parseHex(r.FormValue("seekPos"))

	p, lastMod, seekPos, err := readFileIfModified(lastMod, seekPos, -1)
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
	p, lastMod, fileSize, err := readFileIfModified(time.Time{}, -600, -1)
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
	if err := http.ListenAndServe(":" + *port, nil); err != nil {
		log.Fatal(err)
	}
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<title>{{.LogFileName}}</title>
<style>
.pre-wrap {
	 white-space: pre-wrap;
}
button {
	padding:3px 50px;
}
</style>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.1.1/jquery.min.js"></script>
</head>
<body>
	<pre id="fileDataPre">{{.Data}}</pre>
	<input type="checkbox" id="toggleWrapCheckbox" checked="checked">自动换行</input>
	<input type="checkbox" id="autoRefreshCheckbox" checked="checked">自动刷新</input>
	<button id="refreshButton">刷新</button>
	<button id="clearButton">清空</button>
<script type="text/javascript">
(function() {
	var seekPos = "{{.SeekPos}}"
	var lastMod = "{{.LastMod}}"

	$('#clearButton').click(function() {
		$('#fileDataPre').empty()
	})

	var tailFunction = function() {
		$.ajax({
			type: 'GET',
			url: "/tail",
			data: {
				seekPos: seekPos,
				lastMod: lastMod
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
})()
</script>
</body>
</html>
`
