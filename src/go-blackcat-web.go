package main

import (
	"log"

	"github.com/gocql/gocql"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"strings"
)

func main() {
	app := iris.New()
	app.Adapt(httprouter.New())

	app.Get("/", IndexHandler)
	app.Get("/query/:traceids", BlackcatHandler)
	app.Listen(":8181")
}

func BlackcatHandler(ctx *iris.Context) {
	traceids := ctx.Param("traceids")
	traceidArr := strings.Split(traceids, "X")

	session, _ := CreateSession()
	defer session.Close()

	cql := `SELECT tspretty,linkid,msgtype,msg FROM event_trace WHERE traceid = ?`;

	for _, traceid := range traceidArr {
		ctx.Writef("==========traceid:%s\n", traceid);
		ExecuteQuery(session, cql, traceid, ctx)
	}

}
func CreateSession() (*gocql.Session, error) {
	cluster := gocql.NewCluster("120.77.219.149")
	// cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "blackcat"
	cluster.Consistency = gocql.Quorum
	return cluster.CreateSession()
}

func ExecuteQuery(session *gocql.Session, cql, queryTraceid string, ctx *iris.Context) {
	var tspretty, linkid, msgtype, msg string
	iter := session.Query(cql, queryTraceid).Iter()
	for iter.Scan(&tspretty, &linkid, &msgtype, &msg) {
		ctx.Writef("%s linkId:%s, msgType:%s\n", tspretty, linkid, msgtype)
		ctx.Writef("%s\n\n", msg)
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}

func IndexHandler(ctx *iris.Context) {
	ctx.WriteString(`
<html>
<body>
<textarea placeholder="input traceids, like 210486120201746432X210486119419508736" id="traceids" style="width:1000;">
</textarea>
<br/>

<input type="button" value="Query" onclick="queryLogs()">
<input type="button" value="Clear Result" onclick="clearLogs()">
&nbsp;&nbsp;&nbsp; <input type="checkbox" id="preservResult">Preserve Result
<br/>
<div id="output"></div>
</body>
<script>
var $ = function(id) {
	return document.getElementById(id)
}

function clearLogs() {
	$("output").innerText = ""
}

function queryLogs() {
	minAjax({
		url:"/query/" + $("traceids").value,//request URL
		type:"GET", //Request type GET/POST
		data:{

		},
		success: function(data){
			var old = $("output").innerText
			if ($("preservResult").checked) {
				$("output").innerText = data +  "**************\n"+ old
			} else {
				$("output").innerText = data
			}
		}
	});
}

/*|--minAjax.js--|
  |--(A Minimalistic Pure JavaScript Header for Ajax POST/GET Request )--|
  |--Author : flouthoc (gunnerar7@gmail.com)(http://github.com/flouthoc)--|
  |--Contributers : Add Your Name Below--|
  */
function initXMLhttp() {
	var xmlhttp;
	if (window.XMLHttpRequest) {
		//code for IE7,firefox chrome and above
		xmlhttp = new XMLHttpRequest();
	} else {
		//code for Internet Explorer
		xmlhttp = new ActiveXObject("Microsoft.XMLHTTP");
	}

	return xmlhttp;
}

function minAjax(config) {
	/*
	Config Structure
	url:"reqesting URL"
	type:"GET or POST"
	method: "(OPTIONAL) True for async and False for Non-async | By default its Async"
	debugLog: "(OPTIONAL)To display Debug Logs | By default it is false"
	data: "(OPTIONAL) another Nested Object which should contains reqested Properties in form of Object Properties"
	success: "(OPTIONAL) Callback function to process after response | function(data,status)"
	*/

	if (!config.method) {
		config.method = true;
	}

	if (!config.debugLog) {
		config.debugLog = false;
	}

	var xmlhttp = initXMLhttp();
	xmlhttp.onreadystatechange = function() {
		if (xmlhttp.readyState == 4 && xmlhttp.status == 200) {
			if (config.success) {
				config.success(xmlhttp.responseText, xmlhttp.readyState);
			}

			if (config.debugLog == true) {
				console.log("SuccessResponse");
				console.log("Response Data:" + xmlhttp.responseText);
			}

		} else {
			if (config.debugLog == true)
				console.log("FailureResponse --> State:" + xmlhttp.readyState + "Status:" + xmlhttp.status);
		}
	}

	var sendString = [],
		sendData = config.data;
	if( typeof sendData === "string" ){
		var tmpArr = String.prototype.split.call(sendData,'&');
		for(var i = 0, j = tmpArr.length; i < j; i++){
			var datum = tmpArr[i].split('=');
			sendString.push(encodeURIComponent(datum[0]) + "=" + encodeURIComponent(datum[1]));
		}
	}else if( typeof sendData === 'object' && !( sendData instanceof String || (FormData && sendData instanceof FormData) ) ){
		for (var k in sendData) {
			var datum = sendData[k];
			if( Object.prototype.toString.call(datum) == "[object Array]" ){
				for(var i = 0, j = datum.length; i < j; i++) {
						sendString.push(encodeURIComponent(k) + "[]=" + encodeURIComponent(datum[i]));
				}
			}else{
				sendString.push(encodeURIComponent(k) + "=" + encodeURIComponent(datum));
			}
		}
	}
	sendString = sendString.join('&');

	if (config.type == "GET") {
		xmlhttp.open("GET", config.url + "?" + sendString, config.method);
		xmlhttp.send();

		if (config.debugLog == true)
			console.log("GET fired at:" + config.url + "?" + sendString);
	} else if (config.type == "POST") {
		xmlhttp.open("POST", config.url, config.method);
		xmlhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
		xmlhttp.send(sendString);

		if (config.debugLog == true)
			console.log("POST fired at:" + config.url + " || Data:" + sendString);
	}
}
</script>
</html>
	`)
}